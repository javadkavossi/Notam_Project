package briefing

import (
	"fmt"
	"sort"
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
	EffectRunwayUnavailable = "RUNWAY_UNAVAILABLE"
	EffectAerodromeUnusable = "AERODROME_UNUSABLE"
	EffectApproachDegraded  = "APPROACH_DEGRADED"
	EffectNavDegradation    = "NAV_DEGRADATION"
	EffectRouteRestriction  = "ROUTE_RESTRICTION"
	EffectSurfaceCondition  = "SURFACE_CONDITION"
	EffectServiceLimited    = "SERVICE_LIMITED"
	EffectATSReduced        = "ATS_REDUCED"
	EffectObstacleHazard    = "OBSTACLE_HAZARD"
	EffectRescueLimited     = "RESCUE_LIMITED"
	EffectMetDegraded       = "MET_DEGRADED"
	EffectInformationalOnly = "INFORMATIONAL_ONLY"
	EffectNotApplicable     = "NOT_APPLICABLE"
)

// FlightContext بستر پروازِ لازم برای ارزیابی اثر عملیاتی.
type FlightContext struct {
	AircraftCategory string         // JET / TURBOPROP / PISTON
	FlightRules      string         // IFR / VFR
	RunwayCounts     map[string]int // ICAO فرودگاه → تعداد باند فعال

	// تداخل مسیر/ارتفاع فضای هوایی (E5.6+)
	CruiseAltitudeFt     int // fallback ارتفاع سِیر؛ ۰ = نامعلوم
	Route                RouteContext
	WindowFrom, WindowTo time.Time // پنجرهٔ زمانی پرواز (برای تداخل زمانی)
}

// منابع مسیر و ارتفاع (برای شفافیت و تعیین confidence).
const (
	RouteSourceWaypoints   = "WAYPOINTS"
	RouteSourceGreatCircle = "GREAT_CIRCLE_FALLBACK"
	RouteSourceUnknown     = "UNKNOWN_ROUTE"

	AltSourceSegmentProfile = "SEGMENT_PROFILE"
	AltSourceCruiseFixed    = "CRUISE_FIXED"
	AltSourceNone           = "NONE"
)

// routeLeg یک بخش هندسی از مسیر (مختصات دو سرِ segment).
type routeLeg struct {
	fromSeq, toSeq         int
	lon1, lat1, lon2, lat2 float64
}

// legsFromWaypoints از waypointهای معتبر (پس از فیلترِ نامعتبرها و مرتب‌سازی بر اساس ترتیب)
// بخش‌های مسیر را می‌سازد. اگر کمتر از ۲ نقطهٔ معتبر بماند، nil (fallback).
func legsFromWaypoints(in model.Waypoints) []routeLeg {
	wps := make([]model.Waypoint, 0, len(in))
	for _, w := range in {
		if w.Valid() {
			wps = append(wps, w)
		}
	}
	if len(wps) < 2 {
		return nil
	}
	sort.SliceStable(wps, func(i, j int) bool { return wps[i].Sequence < wps[j].Sequence })
	legs := make([]routeLeg, 0, len(wps)-1)
	for i := 0; i+1 < len(wps); i++ {
		legs = append(legs, routeLeg{
			fromSeq: wps[i].Sequence, toSeq: wps[i+1].Sequence,
			lon1: wps[i].Lon, lat1: wps[i].Lat, lon2: wps[i+1].Lon, lat2: wps[i+1].Lat,
		})
	}
	return legs
}

// SegmentBand یک بخش از مسیر با بازهٔ ارتفاعی و فاز آن.
type SegmentBand struct {
	FromSeq, ToSeq   int
	LowerFt, UpperFt int
	AltKnown         bool
	AltSource        string
	Phase            string
}

// RouteContext مسیر پرواز به‌صورت segmentهای دارای بازهٔ ارتفاعی + نتیجهٔ تقاطع افقی هر segment.
type RouteContext struct {
	Source        string        // WAYPOINTS / GREAT_CIRCLE_FALLBACK / UNKNOWN_ROUTE
	Confidence    string        // HIGH (waypoint کامل) / MEDIUM (great circle)
	Segments      []SegmentBand // به‌ترتیب مسیر
	NotamSegments map[int][]int // notamID → ایندکس segmentهایی که افقی تداخل دارند
}

// وضعیت تداخل هندسی/ارتفاعی/زمانی فضای هوایی (E5.6).
const (
	CtxFullIntersection   = "FULL_INTERSECTION"
	CtxNoIntersection     = "NO_INTERSECTION"
	CtxHorizontalOnly     = "HORIZONTAL_INTERSECTION_ONLY"
	CtxVerticalOnly       = "VERTICAL_INTERSECTION_ONLY"
	CtxUnknownGeometry    = "UNKNOWN_GEOMETRY"
	CtxUnknownAltitude    = "UNKNOWN_ALTITUDE"
	CtxUnknownFlightLevel = "UNKNOWN_FLIGHT_LEVEL"
)

// سطوح اطمینان.
const (
	ConfHigh   = "HIGH"
	ConfMedium = "MEDIUM"
	ConfLow    = "LOW"
)

// MatchedSegment یک segment از مسیر که با محدودهٔ NOTAM تداخل دارد.
type MatchedSegment struct {
	FromSequence           int    `json:"fromSequence"`
	ToSequence             int    `json:"toSequence"`
	Phase                  string `json:"phase,omitempty"`
	HorizontalIntersection bool   `json:"horizontalIntersection"`
	VerticalIntersection   bool   `json:"verticalIntersection"`
}

// GeoAssessment خروجی توضیح‌پذیرِ ارزیابی تداخل فضای هوایی (برای NOTAMهای Airspace/Restriction).
type GeoAssessment struct {
	ContextResult          string           `json:"contextResult"`
	HorizontalIntersection *bool            `json:"horizontalIntersection"` // nil = نامعلوم
	VerticalIntersection   *bool            `json:"verticalIntersection"`   // nil = نامعلوم
	TemporalIntersection   bool             `json:"temporalIntersection"`
	Confidence             string           `json:"confidence"` // HIGH/MEDIUM/LOW
	RouteSource            string           `json:"routeSource"`
	AltitudeSource         string           `json:"altitudeSource"`
	MatchedSegments        []MatchedSegment `json:"matchedSegments,omitempty"`
	EvaluatedSegmentCount  int              `json:"evaluatedSegmentCount"`
	Reasons                []string         `json:"reasons"`
	MissingData            []string         `json:"missingData"`
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
	g := &GeoAssessment{
		TemporalIntersection:  true,
		RouteSource:           ctx.Route.Source,
		EvaluatedSegmentCount: len(ctx.Route.Segments),
	}
	base := n.BaseScore + roleBonus(n, model.RoleEnroute)

	// ---- تداخل زمانی (پنجرهٔ پرواز) ----
	if !ctx.WindowFrom.IsZero() {
		if !temporalOverlap(n, ctx.WindowFrom, ctx.WindowTo) {
			g.TemporalIntersection = false
			g.ContextResult = CtxNoIntersection
			g.Confidence = ConfHigh
			g.Reasons = append(g.Reasons, "زمان پرواز خارج از بازهٔ اعتبار NOTAM است")
			return airspaceResult(informationalScore(n.BaseScore), EffectNotApplicable,
				"NOTAM در زمان این پرواز فعال نیست", g)
		}
	}

	// ---- هندسه یا مسیر نامعلوم → بدون تصمیم قطعی، بدون تشدید ----
	if n.AreaRadiusNM <= 0 {
		g.ContextResult = CtxUnknownGeometry
		g.Confidence = ConfLow
		g.MissingData = append(g.MissingData, "notamGeometry")
		g.Reasons = append(g.Reasons, "هندسهٔ محدودهٔ NOTAM موجود نیست؛ تداخل قابل تأیید نیست")
		return airspaceResult(minInt(base, mediumCap), EffectRouteRestriction, defaultAction(EffectRouteRestriction), g)
	}
	if ctx.Route.Source == RouteSourceUnknown || len(ctx.Route.Segments) == 0 {
		g.ContextResult = CtxUnknownGeometry
		g.Confidence = ConfLow
		g.MissingData = append(g.MissingData, "flightRoute")
		g.Reasons = append(g.Reasons, "مسیر پرواز قابل ساخت نیست (مختصات مبدأ/مقصد یا waypoint ناقص)")
		return airspaceResult(minInt(base, mediumCap), EffectRouteRestriction, defaultAction(EffectRouteRestriction), g)
	}

	// ---- تداخل افقی: کدام segmentها با محدوده تداخل دارند؟ ----
	segIdxs := ctx.Route.NotamSegments[n.Id]
	horizontal := len(segIdxs) > 0
	g.HorizontalIntersection = &horizontal

	if !horizontal {
		g.ContextResult = CtxNoIntersection
		g.Confidence = ctx.Route.Confidence // waypoint→HIGH ، great-circle→MEDIUM
		g.Reasons = append(g.Reasons, routeMissReason(ctx.Route.Source))
		return airspaceResult(informationalScore(n.BaseScore), EffectNotApplicable,
			"مسیر پرواز وارد این محدوده نمی‌شود", g)
	}

	// ---- برای هر segmentِ متقاطع، تداخل عمودی را بسنج ----
	notamAltKnown := n.VerticalKnown
	var matched []MatchedSegment
	anyFull := false
	anyFlightAltUnknown := false
	altSource := AltSourceNone

	for _, si := range segIdxs {
		if si < 0 || si >= len(ctx.Route.Segments) {
			continue
		}
		seg := ctx.Route.Segments[si]
		ms := MatchedSegment{FromSequence: seg.FromSeq, ToSequence: seg.ToSeq, Phase: seg.Phase, HorizontalIntersection: true}
		if seg.AltSource != AltSourceNone {
			altSource = seg.AltSource
		}
		if notamAltKnown && seg.AltKnown {
			// هم‌پوشانی بازهٔ ارتفاعیِ segment با بازهٔ NOTAM
			v := seg.LowerFt <= n.UpperFt && seg.UpperFt >= n.LowerFt
			ms.VerticalIntersection = v
			if v {
				anyFull = true
			}
		} else if notamAltKnown && !seg.AltKnown {
			anyFlightAltUnknown = true
		}
		matched = append(matched, ms)
	}
	g.AltitudeSource = altSource
	g.MatchedSegments = matched

	// ---- ارتفاع NOTAM نامعلوم → بدون تشدید ----
	if !notamAltKnown {
		g.ContextResult = CtxUnknownAltitude
		g.Confidence = ConfLow
		g.MissingData = append(g.MissingData, "notamAltitude")
		g.Reasons = append(g.Reasons, "تداخل افقی هست ولی حدود ارتفاعی NOTAM نامشخص است")
		return airspaceResult(minInt(base, mediumCap), EffectRouteRestriction,
			"تداخل افقی؛ حدود ارتفاعی نامشخص — دستی بررسی شود", g)
	}

	// ---- تداخل کامل: حداقل یک segment با هر سه شرط (افقی+عمودی+زمانی) ----
	if anyFull {
		vTrue := true
		g.VerticalIntersection = &vTrue
		g.ContextResult = CtxFullIntersection
		g.Confidence = ctx.Route.Confidence // عمودی/زمانی دقیق؛ افقی تعیین‌کنندهٔ confidence
		g.Reasons = append(g.Reasons, fullIntersectionReason(matched, n))
		return airspaceResult(analysis.Clamp(base), EffectRouteRestriction,
			"محدودیت فضای هوایی روی مسیر و ارتفاع این پرواز — بررسی و هماهنگی لازم", g)
	}

	// ---- تداخل افقی بود ولی ارتفاع پروازِ بعضی segmentها نامعلوم ----
	if anyFlightAltUnknown {
		g.ContextResult = CtxUnknownFlightLevel
		g.Confidence = ConfLow
		g.MissingData = append(g.MissingData, "flightLevel")
		g.Reasons = append(g.Reasons, "تداخل افقی هست ولی ارتفاع پروازِ بخشی از مسیر ثبت نشده")
		return airspaceResult(minInt(base, mediumCap), EffectRouteRestriction,
			"تداخل افقی؛ ارتفاع پرواز را کامل کنید تا تداخل عمودی سنجیده شود", g)
	}

	// ---- افقی بله، عمودی خیر در همهٔ segmentها ----
	vFalse := false
	g.VerticalIntersection = &vFalse
	g.ContextResult = CtxHorizontalOnly
	g.Confidence = ctx.Route.Confidence
	g.Reasons = append(g.Reasons, fmt.Sprintf("مسیر عبور می‌کند ولی در هیچ بخشی ارتفاع با بازهٔ NOTAM (%d→%dft) هم‌پوشانی ندارد",
		n.LowerFt, n.UpperFt))
	return airspaceResult(informationalScore(n.BaseScore), EffectNotApplicable,
		"مسیر عبور می‌کند ولی در ارتفاع دیگری", g)
}

func routeMissReason(source string) string {
	if source == RouteSourceWaypoints {
		return "مسیرِ واقعی (waypoint) از محدودهٔ NOTAM عبور نمی‌کند"
	}
	return fmt.Sprintf("مسیر از محدودهٔ NOTAM عبور نمی‌کند (کریدور ~%.0fNM، مسیر مستقیم تقریبی)", routeCorridorNM)
}

func fullIntersectionReason(matched []MatchedSegment, n model.Notam) string {
	for _, m := range matched {
		if m.VerticalIntersection {
			phase := m.Phase
			if phase == "" {
				phase = model.PhaseUnknown
			}
			return fmt.Sprintf("Airspace intersects route segment %d→%d during %s within FL band %d→%dft",
				m.FromSequence, m.ToSequence, phase, n.LowerFt, n.UpperFt)
		}
	}
	return "تداخل افقی، عمودی و زمانی تأیید شد"
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
