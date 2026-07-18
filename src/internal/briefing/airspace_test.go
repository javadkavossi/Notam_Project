package briefing

import (
	"testing"
	"time"

	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/pipeline/analysis"
	"github.com/hossein-repo/BaseProject/internal/pipeline/qcode"
)

// airspaceNotam یک NOTAM محدودیت فضای هوایی با هندسه و ارتفاعِ اختیاری می‌سازد.
func airspaceNotam(id int, hasArea bool, lowerFt, upperFt int, vertKnown bool) model.Notam {
	n := model.Notam{
		BaseModel:      model.BaseModel{Id: id},
		AffectedFIR:    "OIIX",
		Category:       qcode.CatRestriction,
		Tags:           model.StringSlice{},
		BaseScore:      60,
		BaseLevel:      analysis.LevelFor(60),
		EffectiveStart: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		VerticalKnown:  vertKnown,
		LowerFt:        lowerFt,
		UpperFt:        upperFt,
	}
	if hasArea {
		n.AreaLat, n.AreaLon, n.AreaRadiusNM = 35.0, 51.0, 20
	}
	return n
}

// بستر پرواز کامل: مسیر معلوم، ارتفاع سِیر معلوم، پنجرهٔ زمانی معلوم.
func fullCtx(intersects bool, notamID int, cruiseFt int) FlightContext {
	return FlightContext{
		FlightRules:      model.RulesIFR,
		AircraftCategory: model.AircraftJet,
		CruiseAltitudeFt: cruiseFt,
		RouteKnown:       true,
		RouteIntersects:  map[int]bool{notamID: intersects},
		WindowFrom:       time.Date(2026, 7, 20, 6, 0, 0, 0, time.UTC),
		WindowTo:         time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC),
	}
}

// ۱) مسیر عبور می‌کند و ارتفاع تداخل دارد → FULL_INTERSECTION → IMPACTED (امتیاز حفظ)
func TestAirspace_FullIntersection(t *testing.T) {
	n := airspaceNotam(1, true, 0, 40000, true) // GND→FL400
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", fullCtx(true, 1, 35000))
	if r.Geo.ContextResult != CtxFullIntersection {
		t.Fatalf("contextResult=%s، انتظار FULL_INTERSECTION", r.Geo.ContextResult)
	}
	if r.Effect != EffectRouteRestriction {
		t.Errorf("اثر باید ROUTE_RESTRICTION بماند، دریافت %s", r.Effect)
	}
	if r.Score < n.BaseScore {
		t.Errorf("تداخل واقعی نباید امتیاز را کم کند: %d < %d", r.Score, n.BaseScore)
	}
	if *r.Geo.HorizontalIntersection != true || *r.Geo.VerticalIntersection != true {
		t.Error("هر دو تداخل افقی و عمودی باید true باشند")
	}
}

// ۲) مسیر عبور می‌کند ولی ارتفاع خارج از محدوده → HORIZONTAL_INTERSECTION_ONLY → NOT_APPLICABLE
func TestAirspace_HorizontalOnly(t *testing.T) {
	n := airspaceNotam(2, true, 0, 5000, true) // GND→FL050
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", fullCtx(true, 2, 35000))
	if r.Geo.ContextResult != CtxHorizontalOnly {
		t.Fatalf("contextResult=%s، انتظار HORIZONTAL_INTERSECTION_ONLY", r.Geo.ContextResult)
	}
	if r.Effect != EffectNotApplicable {
		t.Errorf("باید NOT_APPLICABLE شود، دریافت %s", r.Effect)
	}
	if r.Level == analysis.LevelCritical || r.Level == analysis.LevelHigh {
		t.Errorf("نباید بحرانی/بالا باشد (%s %d)", r.Level, r.Score)
	}
}

// ۳) ارتفاع تداخل دارد ولی مسیر عبور نمی‌کند → VERTICAL_INTERSECTION_ONLY → NOT_APPLICABLE
func TestAirspace_VerticalOnly(t *testing.T) {
	n := airspaceNotam(3, true, 0, 40000, true)
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", fullCtx(false, 3, 35000))
	// مسیر عبور نمی‌کند → دروازهٔ افقی: NO_INTERSECTION (ارتفاع بی‌اهمیت)
	if r.Geo.ContextResult != CtxNoIntersection {
		t.Fatalf("contextResult=%s، انتظار NO_INTERSECTION (مسیر عبور نمی‌کند)", r.Geo.ContextResult)
	}
	if r.Effect != EffectNotApplicable {
		t.Errorf("باید NOT_APPLICABLE شود، دریافت %s", r.Effect)
	}
}

// ۴) زمان پرواز خارج از بازهٔ NOTAM → NO_INTERSECTION (زمانی) → NOT_APPLICABLE
func TestAirspace_TemporalOutside(t *testing.T) {
	n := airspaceNotam(4, true, 0, 40000, true)
	end := time.Date(2026, 7, 19, 0, 0, 0, 0, time.UTC) // پیش از پنجرهٔ پرواز
	n.EffectiveStart = time.Date(2026, 7, 18, 0, 0, 0, 0, time.UTC)
	n.EffectiveEnd = &end
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", fullCtx(true, 4, 35000))
	if r.Geo.TemporalIntersection {
		t.Error("temporalIntersection باید false باشد")
	}
	if r.Effect != EffectNotApplicable {
		t.Errorf("خارج از بازهٔ زمانی باید NOT_APPLICABLE شود، دریافت %s", r.Effect)
	}
}

// ۵) geometry موجود نیست → UNKNOWN_GEOMETRY، بدون تشدید
func TestAirspace_UnknownGeometry(t *testing.T) {
	n := airspaceNotam(5, false, 0, 40000, true) // بدون area
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", fullCtx(true, 5, 35000))
	if r.Geo.ContextResult != CtxUnknownGeometry {
		t.Fatalf("contextResult=%s، انتظار UNKNOWN_GEOMETRY", r.Geo.ContextResult)
	}
	if r.Level == analysis.LevelCritical || r.Level == analysis.LevelHigh {
		t.Errorf("نبود هندسه نباید HIGH/CRITICAL شود (%s %d)", r.Level, r.Score)
	}
	if len(r.Geo.MissingData) == 0 {
		t.Error("missingData باید پر شود")
	}
	if r.Geo.Confidence != ConfLow {
		t.Errorf("confidence باید LOW باشد، دریافت %s", r.Geo.Confidence)
	}
}

// ۶) ارتفاع NOTAM نامشخص → UNKNOWN_ALTITUDE، بدون تشدید
func TestAirspace_UnknownNotamAltitude(t *testing.T) {
	n := airspaceNotam(6, true, 0, 0, false) // vertical unknown
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", fullCtx(true, 6, 35000))
	if r.Geo.ContextResult != CtxUnknownAltitude {
		t.Fatalf("contextResult=%s، انتظار UNKNOWN_ALTITUDE", r.Geo.ContextResult)
	}
	if r.Level == analysis.LevelCritical || r.Level == analysis.LevelHigh {
		t.Errorf("نبود ارتفاع NOTAM نباید HIGH/CRITICAL شود (%s %d)", r.Level, r.Score)
	}
	if r.Geo.VerticalIntersection != nil {
		t.Error("verticalIntersection باید nil (نامعلوم) باشد")
	}
}

// ۷) ارتفاع پرواز نامشخص → UNKNOWN_FLIGHT_LEVEL
func TestAirspace_UnknownFlightLevel(t *testing.T) {
	n := airspaceNotam(7, true, 0, 40000, true)
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", fullCtx(true, 7, 0)) // cruise=0 نامعلوم
	if r.Geo.ContextResult != CtxUnknownFlightLevel {
		t.Fatalf("contextResult=%s، انتظار UNKNOWN_FLIGHT_LEVEL", r.Geo.ContextResult)
	}
	if r.Level == analysis.LevelCritical || r.Level == analysis.LevelHigh {
		t.Errorf("نبود ارتفاع پرواز نباید HIGH/CRITICAL شود (%s %d)", r.Level, r.Score)
	}
	found := false
	for _, m := range r.Geo.MissingData {
		if m == "flightLevel" {
			found = true
		}
	}
	if !found {
		t.Errorf("missingData باید شامل flightLevel باشد: %v", r.Geo.MissingData)
	}
}

// ۸) مسیر مماس با محدوده (تداخل مرزی) — PostGIS مماس را تداخل می‌شمارد → FULL_INTERSECTION
func TestAirspace_Tangent(t *testing.T) {
	n := airspaceNotam(8, true, 10000, 40000, true)
	// مماس = intersects=true از نظر ST_Intersects
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", fullCtx(true, 8, 35000))
	if r.Geo.ContextResult != CtxFullIntersection {
		t.Errorf("مماس با ارتفاعِ داخلِ بازه باید FULL_INTERSECTION شود، دریافت %s", r.Geo.ContextResult)
	}
}

// ۹) مسیر چندبخشی که فقط یک segment تداخل دارد — از منظر PostGIS، RouteIntersects=true کافی است
func TestAirspace_MultiSegmentOneHits(t *testing.T) {
	// شبیه‌سازی: کوئری فضایی intersects=true برگردانده (یک segment برخورد کرده)
	n := airspaceNotam(9, true, 0, 40000, true)
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", fullCtx(true, 9, 20000))
	if r.Geo.ContextResult != CtxFullIntersection {
		t.Errorf("اگر هر بخشی از مسیر تداخل کند باید FULL_INTERSECTION شود، دریافت %s", r.Geo.ContextResult)
	}
}

// ۱۰) NOTAM بدون تداخل واقعی روی مسیر → NOT_APPLICABLE (مثال کاهش HIGH→NOT_APPLICABLE)
func TestAirspace_NoIntersectionDowngrades(t *testing.T) {
	n := airspaceNotam(10, true, 0, 40000, true)
	n.BaseScore = 67 // در حالت پایه HIGH بود
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", fullCtx(false, 10, 35000))
	if r.Geo.ContextResult != CtxNoIntersection {
		t.Fatalf("contextResult=%s، انتظار NO_INTERSECTION", r.Geo.ContextResult)
	}
	if r.Effect != EffectNotApplicable {
		t.Errorf("باید NOT_APPLICABLE شود، دریافت %s", r.Effect)
	}
	if r.Score >= 35 {
		t.Errorf("HIGH پایه باید به امتیاز پایین کاهش یابد، دریافت %d", r.Score)
	}
}

// اصل ایمنی صریح: هیچ‌یک از حالت‌های نامعلوم نباید HIGH/CRITICAL تولید کند.
func TestAirspace_UnknownsNeverEscalate(t *testing.T) {
	cases := []struct {
		name string
		n    model.Notam
		ctx  FlightContext
	}{
		{"no-geometry", airspaceNotam(20, false, 0, 40000, true), fullCtx(true, 20, 35000)},
		{"no-route", airspaceNotam(21, true, 0, 40000, true), FlightContext{RouteKnown: false, WindowFrom: time.Date(2026, 7, 20, 6, 0, 0, 0, time.UTC), WindowTo: time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)}},
		{"no-notam-alt", airspaceNotam(22, true, 0, 0, false), fullCtx(true, 22, 35000)},
		{"no-flight-level", airspaceNotam(23, true, 0, 40000, true), fullCtx(true, 23, 0)},
	}
	for _, c := range cases {
		r := EvaluateImpact(c.n, model.RoleEnroute, "OIIX", c.ctx)
		if r.Level == analysis.LevelCritical || r.Level == analysis.LevelHigh {
			t.Errorf("[%s] حالت نامعلوم نباید %s شود (score=%d)", c.name, r.Level, r.Score)
		}
	}
}
