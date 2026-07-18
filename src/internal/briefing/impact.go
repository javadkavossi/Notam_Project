package briefing

import (
	"fmt"
	"strings"
	"time"

	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/pipeline/analysis"
	"github.com/hossein-repo/BaseProject/internal/pipeline/qcode"
)

// impact.go — لایهٔ Operational Impact (E5.5).
// امتیاز پایهٔ مستقل از پرواز را با بستر پرواز تعدیل می‌کند تا امتیاز نهایی، اثر عملیاتی و
// اقدام پیشنهادی به‌دست آید. منطق خالص است (بدون DB) تا واحد‌تست شود.
// مبنای طراحی: بازخورد کارشناس دیسپچ — docs/phase1/ANALYSIS_AND_BRIEFING.md §۱۰.

// اثرهای عملیاتی استاندارد (چه بلایی سر پرواز می‌آورد، فارغ از اینکه چه تأسیساتی است).
const (
	EffectRunwayUnavailable  = "RUNWAY_UNAVAILABLE"
	EffectAerodromeUnusable  = "AERODROME_UNUSABLE"
	EffectApproachDegraded   = "APPROACH_DEGRADED"
	EffectNavDegradation     = "NAV_DEGRADATION"
	EffectRouteRestriction   = "ROUTE_RESTRICTION"
	EffectSurfaceCondition   = "SURFACE_CONDITION"
	EffectServiceLimited     = "SERVICE_LIMITED"
	EffectATSReduced         = "ATS_REDUCED"
	EffectObstacleHazard     = "OBSTACLE_HAZARD"
	EffectRescueLimited      = "RESCUE_LIMITED"
	EffectMetDegraded        = "MET_DEGRADED"
	EffectInformationalOnly  = "INFORMATIONAL_ONLY"
	EffectNotApplicable      = "NOT_APPLICABLE"
)

// FlightContext بستر پروازِ لازم برای ارزیابی اثر عملیاتی.
type FlightContext struct {
	AircraftCategory string         // JET / TURBOPROP / PISTON
	FlightRules      string         // IFR / VFR
	RunwayCounts     map[string]int // ICAO فرودگاه → تعداد باند فعال

	// تداخل مسیر/ارتفاع فضای هوایی (E5.6)
	CruiseAltitudeFt     int          // ۰ = نامعلوم → UNKNOWN_FLIGHT_LEVEL
	RouteKnown           bool         // آیا کریدور مسیر ساخته شد (مختصات مبدأ/مقصد موجود)؟
	RouteIntersects      map[int]bool // notamID → area با کریدور مسیر تداخل افقی دارد؟
	WindowFrom, WindowTo time.Time    // پنجرهٔ زمانی پرواز (برای تداخل زمانی)
}

// وضعیت تداخل هندسی/ارتفاعی/زمانی فضای هوایی (E5.6).
const (
	CtxFullIntersection    = "FULL_INTERSECTION"
	CtxNoIntersection      = "NO_INTERSECTION"
	CtxHorizontalOnly      = "HORIZONTAL_INTERSECTION_ONLY"
	CtxVerticalOnly        = "VERTICAL_INTERSECTION_ONLY"
	CtxUnknownGeometry     = "UNKNOWN_GEOMETRY"
	CtxUnknownAltitude     = "UNKNOWN_ALTITUDE"
	CtxUnknownFlightLevel  = "UNKNOWN_FLIGHT_LEVEL"
)

// سطوح اطمینان.
const (
	ConfHigh   = "HIGH"
	ConfMedium = "MEDIUM"
	ConfLow    = "LOW"
)

// GeoAssessment خروجی توضیح‌پذیرِ ارزیابی تداخل فضای هوایی (برای NOTAMهای Airspace/Restriction).
type GeoAssessment struct {
	ContextResult          string   `json:"contextResult"`
	HorizontalIntersection *bool    `json:"horizontalIntersection"` // nil = نامعلوم
	VerticalIntersection   *bool    `json:"verticalIntersection"`   // nil = نامعلوم
	TemporalIntersection   bool     `json:"temporalIntersection"`
	Confidence             string   `json:"confidence"` // HIGH/MEDIUM/LOW
	Reasons                []string `json:"reasons"`
	MissingData            []string `json:"missingData"`
}

// ImpactResult خروجی ارزیابی اثر عملیاتی.
type ImpactResult struct {
	Score  int
	Level  string
	Effect string
	Action string         // اقدام پیشنهادی برای خلبان/دیسپچر
	Geo    *GeoAssessment // فقط برای فضای هوایی (E5.6)
}

// EvaluateImpact امتیاز نهایی + اثر + اقدام را برای یک NOTAM در بستر یک پرواز می‌سازد.
func EvaluateImpact(n model.Notam, role, roleICAO string, ctx FlightContext) ImpactResult {
	effect := operationalEffect(n)

	// ---- فاکتور بستر ۱: قوانین پرواز (IFR/VFR) — «بسته به VFR» برای پرواز IFR بی‌اثر است ----
	if notApplicableByRules(n, ctx.FlightRules) {
		return notApplicable(n.BaseScore, "برای قوانین پرواز این سفر ("+ctx.FlightRules+") کاربرد ندارد")
	}

	// ---- فاکتور بستر ۲: بی‌ربطی سرویس (نوع سوخت با نوع هواپیما) ----
	if effect == EffectServiceLimited {
		if fuel := fuelTypeInText(n.PlainText); fuel != "" && !fuelMatchesAircraft(fuel, ctx.AircraftCategory) {
			return notApplicable(n.BaseScore, "سوخت "+fuel+" برای این هواپیما ("+ctx.AircraftCategory+") کاربرد ندارد")
		}
	}

	// ---- فاکتور بستر: تداخل مسیر/ارتفاع/زمان برای فضای هوایی (E5.6) ----
	if effect == EffectRouteRestriction {
		return assessAirspace(n, ctx)
	}

	// ---- فاکتور بستر ۳: تعداد باند — سیگنالِ غالب برای بستن باند ----
	// طبق کارشناس: «اگر تک‌باند ≈۱۰۰، وگرنه ≈۷۰». اینجا roleBonus عمومی جمع نمی‌شود
	// تا با سیگنال باند دوباره‌شماری نکند.
	if effect == EffectRunwayUnavailable && isAirportRole(role) {
		if rc, known := ctx.RunwayCounts[strings.ToUpper(roleICAO)]; known {
			switch {
			case rc <= 1:
				return result(maxInt(n.BaseScore, 92), EffectAerodromeUnusable,
					"فرودگاه عملاً تک‌باند است؛ بستن باند یعنی نبود گزینهٔ فرود — الترنت را بررسی کن")
			case rc == 2:
				return result(minInt(n.BaseScore, 78), effect,
					"یک باند از دو باند بسته است؛ ظرفیت و باندِ موردِ استفاده را بررسی کن")
			default: // rc >= 3
				return result(minInt(n.BaseScore, 66), effect,
					"یکی از چند باند بسته است؛ معمولاً باند جایگزین موجود است")
			}
		}
	}

	// حالت عمومی: امتیاز پایه + تعدیل نقش فرودگاه
	score := analysis.Clamp(n.BaseScore + roleBonus(n, role))
	return result(score, effect, defaultAction(effect))
}

func result(score int, effect, action string) ImpactResult {
	score = analysis.Clamp(score)
	return ImpactResult{Score: score, Level: analysis.LevelFor(score), Effect: effect, Action: action}
}

// medianCap سقفِ MEDIUM؛ برای موارد نامعلوم تا هرگز به‌طور خودکار HIGH/CRITICAL نشوند.
const mediumCap = 59

// assessAirspace تداخل افقی/عمودی/زمانی یک NOTAM فضای هوایی را با پرواز ارزیابی می‌کند (E5.6).
//
// اصل ایمنی: نبودِ geometry/altitude/flight-level هرگز به HIGH/CRITICAL نگاشت نمی‌شود؛
// IMPACTED فقط با تداخلِ تأییدشده صادر می‌شود. کم‌گویی هم پرهیز می‌شود: موارد نامعلوم پنهان
// نمی‌شوند (تا سطح MEDIUM قابل‌مشاهده می‌مانند و در missingData علامت می‌خورند).
func assessAirspace(n model.Notam, ctx FlightContext) ImpactResult {
	g := &GeoAssessment{TemporalIntersection: true}
	base := n.BaseScore + roleBonus(n, model.RoleEnroute)

	// ---- تداخل زمانی (پنجرهٔ پرواز) ----
	if !ctx.WindowFrom.IsZero() {
		temporal := temporalOverlap(n, ctx.WindowFrom, ctx.WindowTo)
		g.TemporalIntersection = temporal
		if !temporal {
			g.ContextResult = CtxNoIntersection
			g.Confidence = ConfHigh
			g.Reasons = append(g.Reasons, "زمان پرواز خارج از بازهٔ اعتبار NOTAM است")
			return airspaceResult(informationalScore(n.BaseScore), EffectNotApplicable,
				"NOTAM در زمان این پرواز فعال نیست", g)
		}
	}

	hasArea := n.AreaRadiusNM > 0
	notamAltKnown := n.VerticalKnown
	flightAltKnown := ctx.CruiseAltitudeFt > 0

	// ---- هندسه یا مسیر نامعلوم → بدون تصمیم قطعی، بدون تشدید ----
	if !hasArea {
		g.ContextResult = CtxUnknownGeometry
		g.Confidence = ConfLow
		g.MissingData = append(g.MissingData, "notamGeometry")
		g.Reasons = append(g.Reasons, "هندسهٔ محدودهٔ NOTAM موجود نیست؛ تداخل قابل تأیید نیست")
		return airspaceResult(minInt(base, mediumCap), EffectRouteRestriction, defaultAction(EffectRouteRestriction), g)
	}
	if !ctx.RouteKnown {
		g.ContextResult = CtxUnknownGeometry
		g.Confidence = ConfLow
		g.MissingData = append(g.MissingData, "flightRoute")
		g.Reasons = append(g.Reasons, "مختصات مبدأ/مقصد برای ساخت مسیر موجود نیست")
		return airspaceResult(minInt(base, mediumCap), EffectRouteRestriction, defaultAction(EffectRouteRestriction), g)
	}

	// ---- تداخل افقی (مسیر ↔ محدوده) ----
	horizontal := ctx.RouteIntersects[n.Id]
	g.HorizontalIntersection = &horizontal

	// اگر مسیر عبور نمی‌کند، ارتفاع بی‌اهمیت است (دروازهٔ افقی قطعی).
	if !horizontal {
		g.ContextResult = CtxNoIntersection
		g.Confidence = ConfMedium // مسیرِ مستقیمِ تقریبی
		g.Reasons = append(g.Reasons, fmt.Sprintf("مسیر از محدودهٔ NOTAM عبور نمی‌کند (کریدور ~%.0fNM، مسیر مستقیم تقریبی)", routeCorridorNM))
		return airspaceResult(informationalScore(n.BaseScore), EffectNotApplicable,
			"مسیر پرواز وارد این محدوده نمی‌شود", g)
	}

	// ---- مسیر عبور می‌کند → بررسی ارتفاع ----
	if !notamAltKnown {
		g.ContextResult = CtxUnknownAltitude
		g.Confidence = ConfLow
		g.MissingData = append(g.MissingData, "notamAltitude")
		g.Reasons = append(g.Reasons, "تداخل افقی هست ولی حدود ارتفاعی NOTAM نامشخص است")
		return airspaceResult(minInt(base, mediumCap), EffectRouteRestriction,
			"تداخل افقی؛ حدود ارتفاعی نامشخص — دستی بررسی شود", g)
	}
	if !flightAltKnown {
		g.ContextResult = CtxUnknownFlightLevel
		g.Confidence = ConfLow
		g.MissingData = append(g.MissingData, "flightLevel")
		g.Reasons = append(g.Reasons, "تداخل افقی هست ولی ارتفاع سِیر پرواز ثبت نشده")
		return airspaceResult(minInt(base, mediumCap), EffectRouteRestriction,
			"تداخل افقی؛ ارتفاع پرواز را وارد کنید تا تداخل عمودی سنجیده شود", g)
	}

	vertical := ctx.CruiseAltitudeFt >= n.LowerFt && ctx.CruiseAltitudeFt <= n.UpperFt
	g.VerticalIntersection = &vertical
	g.Confidence = ConfMedium // افقی تقریبی، عمودی/زمانی دقیق

	if vertical {
		g.ContextResult = CtxFullIntersection
		g.Reasons = append(g.Reasons, fmt.Sprintf("تداخل افقی، عمودی (%dft در بازهٔ %d→%dft) و زمانی تأیید شد",
			ctx.CruiseAltitudeFt, n.LowerFt, n.UpperFt))
		// IMPACTED: امتیاز کامل حفظ می‌شود (تداخل واقعی)
		return airspaceResult(analysis.Clamp(base), EffectRouteRestriction,
			"محدودیت فضای هوایی روی مسیر و ارتفاع این پرواز — بررسی و هماهنگی لازم", g)
	}

	// تداخل افقی هست ولی ارتفاع خارج از بازه
	g.ContextResult = CtxHorizontalOnly
	g.Reasons = append(g.Reasons, fmt.Sprintf("مسیر عبور می‌کند ولی ارتفاع پرواز (%dft) خارج از بازهٔ NOTAM (%d→%dft) است",
		ctx.CruiseAltitudeFt, n.LowerFt, n.UpperFt))
	return airspaceResult(informationalScore(n.BaseScore), EffectNotApplicable,
		"مسیر عبور می‌کند ولی در ارتفاع دیگری", g)
}

func airspaceResult(score int, effect, action string, g *GeoAssessment) ImpactResult {
	score = analysis.Clamp(score)
	return ImpactResult{Score: score, Level: analysis.LevelFor(score), Effect: effect, Action: action, Geo: g}
}

// temporalOverlap: بازهٔ اعتبار NOTAM با پنجرهٔ پرواز هم‌پوشانی دارد؟
func temporalOverlap(n model.Notam, from, to time.Time) bool {
	if !n.EffectiveStart.IsZero() && n.EffectiveStart.After(to) {
		return false
	}
	if n.EffectiveEnd != nil && !n.EffectiveEnd.IsZero() && n.EffectiveEnd.Before(from) {
		return false
	}
	return true
}

func notApplicable(base int, action string) ImpactResult {
	s := informationalScore(base)
	return ImpactResult{Score: s, Level: analysis.LevelFor(s), Effect: EffectNotApplicable, Action: action}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// operationalEffect NOTAM را به یک اثر عملیاتی استاندارد نگاشت می‌کند.
func operationalEffect(n model.Notam) string {
	if hasTag(n.Tags, analysis.TagAdClosed) {
		return EffectAerodromeUnusable
	}
	if hasTag(n.Tags, analysis.TagRwyClosed) {
		return EffectRunwayUnavailable
	}
	if hasTag(n.Tags, analysis.TagFICON) {
		return EffectSurfaceCondition
	}
	if n.QCondition == "TT" {
		return EffectInformationalOnly
	}
	switch n.Category {
	case qcode.CatRunway:
		if isClosed(n) {
			return EffectRunwayUnavailable
		}
		return EffectSurfaceCondition
	case qcode.CatAerodrome:
		if isClosed(n) {
			return EffectAerodromeUnusable
		}
		return EffectServiceLimited
	case qcode.CatILS, qcode.CatLighting, qcode.CatProcedure:
		return EffectApproachDegraded
	case qcode.CatNavigation, qcode.CatGNSS:
		return EffectNavDegradation
	case qcode.CatAirspace, qcode.CatRestriction:
		return EffectRouteRestriction
	case qcode.CatService:
		return EffectServiceLimited
	case qcode.CatATS, qcode.CatComms:
		return EffectATSReduced
	case qcode.CatObstacle:
		return EffectObstacleHazard
	case qcode.CatRescue:
		return EffectRescueLimited
	case qcode.CatMet:
		return EffectMetDegraded
	default:
		return EffectInformationalOnly
	}
}

// roleBonus تعدیل بر اساس نقش فرودگاه در پرواز (منطق قبلی، حفظ‌شده).
func roleBonus(n model.Notam, role string) int {
	switch role {
	case model.RoleADES:
		if arrivalCategories[n.Category] {
			return 12
		}
		return 4
	case model.RoleADEP:
		if departureCategories[n.Category] {
			return 8
		}
		return 3
	case model.RoleALTN:
		if n.Category == qcode.CatAerodrome || n.Category == qcode.CatRunway {
			return 6
		}
		return 2
	case model.RoleEnroute:
		if n.Category == qcode.CatAirspace || n.Category == qcode.CatRestriction || n.Category == qcode.CatNavigation {
			return 5
		}
	}
	return 0
}

// notApplicableByRules: «بسته به VFR» + پرواز IFR (یا برعکس) → بی‌اثر.
func notApplicableByRules(n model.Notam, rules string) bool {
	switch strings.ToUpper(rules) {
	case model.RulesIFR:
		// LV = Closed to VFR operations؛ traffic=V یعنی فقط VFR
		return n.QCondition == "LV" || n.Traffic == "V"
	case model.RulesVFR:
		// LI = Closed to IFR operations؛ traffic=I یعنی فقط IFR
		return n.QCondition == "LI" || n.Traffic == "I"
	}
	return false
}

// fuelTypeInText نوع سوخت ذکرشده در متن NOTAM را تشخیص می‌دهد (اگر باشد).
func fuelTypeInText(text string) string {
	up := strings.ToUpper(text)
	switch {
	case strings.Contains(up, "100LL") || strings.Contains(up, "AVGAS") || strings.Contains(up, "100 LL"):
		return "AVGAS"
	case strings.Contains(up, "JET A1") || strings.Contains(up, "JET A-1") || strings.Contains(up, "JETA1") || strings.Contains(up, "JET-A1") || strings.Contains(up, "JET FUEL") || strings.Contains(up, "JP-8"):
		return "JETA1"
	}
	return ""
}

// fuelMatchesAircraft: هواپیمای جت/توربوپراپ سوخت جت می‌خواهد؛ پیستونی AVGAS.
func fuelMatchesAircraft(fuel, aircraft string) bool {
	switch strings.ToUpper(aircraft) {
	case model.AircraftJet, model.AircraftTurboprop:
		return fuel == "JETA1"
	case model.AircraftPiston:
		return fuel == "AVGAS"
	}
	return true // نامشخص → محتاطانه مرتبط فرض کن
}

func hasTag(tags model.StringSlice, t string) bool {
	for _, x := range tags {
		if x == t {
			return true
		}
	}
	return false
}

func isAirportRole(role string) bool {
	return role == model.RoleADEP || role == model.RoleADES || role == model.RoleALTN
}

func isClosed(n model.Notam) bool {
	if n.QCondition == "LC" {
		return true
	}
	up := strings.ToUpper(n.PlainText)
	return strings.Contains(up, "CLSD") || strings.Contains(up, "CLOSED")
}

// informationalScore امتیاز موارد بی‌ربط را پایین می‌آورد ولی صفر نمی‌کند (هیچ NOTAMی بی‌صاحب نماند).
func informationalScore(base int) int {
	s := base / 4
	if s > 20 {
		s = 20
	}
	return s
}

func defaultAction(effect string) string {
	switch effect {
	case EffectAerodromeUnusable:
		return "فرودگاه غیرقابل‌استفاده — الترنت را بررسی کن"
	case EffectRunwayUnavailable:
		return "باندِ موردِ استفاده و باند جایگزین را بررسی کن"
	case EffectApproachDegraded:
		return "حداقل‌های نزدیکی و رویهٔ جایگزین را بررسی کن"
	case EffectNavDegradation:
		return "اتکای مسیر به این ناوید و جایگزین RNAV را بررسی کن"
	case EffectRouteRestriction:
		return "تداخل مسیر و ارتفاع پرواز با محدودیت را بررسی کن"
	case EffectSurfaceCondition:
		return "وضعیت سطح باند و اثر آن بر فرود/برخاست را بررسی کن"
	case EffectServiceLimited:
		return "نیاز پرواز به این سرویس را بررسی کن"
	case EffectATSReduced:
		return "ساعت عملیات و سرویس جایگزین را بررسی کن"
	case EffectObstacleHazard:
		return "مانع در مسیر تقرب/برخاست را بررسی کن"
	case EffectRescueLimited:
		return "دستهٔ آتش‌نشانی موردنیاز (پیش از پرواز) را بررسی کن"
	case EffectMetDegraded:
		return "اثر بر عملیات کم‌دید (RVR/باد) را بررسی کن"
	default:
		return "اطلاعی"
	}
}
