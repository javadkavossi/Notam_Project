package briefing

import (
	"testing"
	"time"

	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/pipeline/analysis"
	"github.com/hossein-repo/BaseProject/internal/pipeline/qcode"
)

func win() (time.Time, time.Time) {
	return time.Date(2026, 7, 20, 6, 0, 0, 0, time.UTC), time.Date(2026, 7, 20, 12, 0, 0, 0, time.UTC)
}

// ctxRoute یک بستر پرواز با مسیر segment-based می‌سازد.
func ctxRoute(source string, conf string, segs []SegmentBand, notamSegs map[int][]int) FlightContext {
	from, to := win()
	return FlightContext{
		FlightRules: model.RulesIFR, AircraftCategory: model.AircraftJet,
		WindowFrom: from, WindowTo: to,
		Route: RouteContext{Source: source, Confidence: conf, Segments: segs, NotamSegments: notamSegs},
	}
}

func seg(from, to, lower, upper int, phase string) SegmentBand {
	return SegmentBand{FromSeq: from, ToSeq: to, LowerFt: lower, UpperFt: upper, AltKnown: true, AltSource: AltSourceSegmentProfile, Phase: phase}
}

// airNotam یک NOTAM فضای هوایی با هندسه و بازهٔ ارتفاعی.
func airNotam(id, lowerFt, upperFt int) model.Notam {
	from, _ := win()
	return model.Notam{
		BaseModel: model.BaseModel{Id: id}, Category: qcode.CatRestriction,
		BaseScore: 60, BaseLevel: analysis.LevelFor(60),
		EffectiveStart: from.Add(-time.Hour), AreaLat: 35, AreaLon: 51, AreaRadiusNM: 20,
		LowerFt: lowerFt, UpperFt: upperFt, VerticalKnown: true,
	}
}

// ۱) مسیر waypoint واقعی با یک segment متقاطع (ارتفاع داخل بازه) → FULL_INTERSECTION
func TestRoute_WaypointOneSegmentFull(t *testing.T) {
	n := airNotam(1, 0, 40000)
	segs := []SegmentBand{seg(1, 2, 30000, 36000, model.PhaseCruise), seg(2, 3, 34000, 34000, model.PhaseCruise)}
	ctx := ctxRoute(RouteSourceWaypoints, ConfHigh, segs, map[int][]int{1: {0}})
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", ctx)
	if r.Geo.ContextResult != CtxFullIntersection {
		t.Fatalf("انتظار FULL_INTERSECTION، دریافت %s", r.Geo.ContextResult)
	}
	if r.Geo.RouteSource != RouteSourceWaypoints || r.Geo.Confidence != ConfHigh {
		t.Errorf("مسیر واقعی باید confidence=HIGH بدهد: source=%s conf=%s", r.Geo.RouteSource, r.Geo.Confidence)
	}
	if len(r.Geo.MatchedSegments) == 0 || !r.Geo.MatchedSegments[0].VerticalIntersection {
		t.Error("matchedSegments باید segmentِ متقاطع را نشان دهد")
	}
}

// ۲) Great Circle متقاطع ولی waypoint واقعی غیرمتقاطع → نتیجهٔ متفاوت
func TestRoute_GreatCircleFalsePositiveVsWaypoint(t *testing.T) {
	n := airNotam(2, 0, 40000)
	// GC: تک‌segment، تداخل دارد → FULL
	gc := fullCtx(true, 2, 34000)
	if EvaluateImpact(n, model.RoleEnroute, "OIIX", gc).Geo.ContextResult != CtxFullIntersection {
		t.Fatal("GC باید تداخل نشان دهد (مثبت کاذب)")
	}
	// Waypoint واقعی: هیچ segmentی تداخل ندارد → NO_INTERSECTION
	wp := ctxRoute(RouteSourceWaypoints, ConfHigh, []SegmentBand{seg(1, 2, 34000, 34000, model.PhaseCruise)}, map[int][]int{})
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", wp)
	if r.Geo.ContextResult != CtxNoIntersection {
		t.Errorf("مسیر واقعی باید NO_INTERSECTION بدهد، دریافت %s", r.Geo.ContextResult)
	}
	if r.Geo.Confidence != ConfHigh {
		t.Errorf("رد تداخل با مسیر واقعی باید confidence=HIGH باشد، دریافت %s", r.Geo.Confidence)
	}
}

// ۳) تداخل فقط در climb
func TestRoute_ClimbOnly(t *testing.T) {
	n := airNotam(3, 0, 15000) // ناحیهٔ پایین GND→FL150
	segs := []SegmentBand{
		seg(1, 2, 5000, 18000, model.PhaseClimb),   // صعود: با ناحیه هم‌پوشانی دارد
		seg(2, 3, 35000, 35000, model.PhaseCruise), // سِیر: بالاتر
	}
	// فقط segment صعود افقی تداخل دارد
	ctx := ctxRoute(RouteSourceWaypoints, ConfHigh, segs, map[int][]int{3: {0}})
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", ctx)
	if r.Geo.ContextResult != CtxFullIntersection {
		t.Fatalf("انتظار FULL_INTERSECTION در صعود، دریافت %s", r.Geo.ContextResult)
	}
	if r.Geo.MatchedSegments[0].Phase != model.PhaseClimb {
		t.Errorf("فاز باید CLIMB باشد، دریافت %s", r.Geo.MatchedSegments[0].Phase)
	}
}

// ۴) تداخل فقط در descent
func TestRoute_DescentOnly(t *testing.T) {
	n := airNotam(4, 0, 12000)
	segs := []SegmentBand{
		seg(1, 2, 35000, 35000, model.PhaseCruise),
		seg(2, 3, 3000, 14000, model.PhaseDescent), // نزول: هم‌پوشانی
	}
	ctx := ctxRoute(RouteSourceWaypoints, ConfHigh, segs, map[int][]int{4: {1}})
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", ctx)
	if r.Geo.ContextResult != CtxFullIntersection || r.Geo.MatchedSegments[0].Phase != model.PhaseDescent {
		t.Errorf("انتظار FULL در نزول: %s phase=%v", r.Geo.ContextResult, r.Geo.MatchedSegments)
	}
}

// ۵) عدم تداخل در cruise ولی تداخل در climb (که با cruiseAltitude قبلی از دست می‌رفت)
func TestRoute_CruiseMissesButClimbHits(t *testing.T) {
	n := airNotam(5, 0, 10000) // ناحیهٔ پایین
	segs := []SegmentBand{
		seg(1, 2, 2000, 12000, model.PhaseClimb),   // صعود: هم‌پوشانی دارد
		seg(2, 3, 35000, 35000, model.PhaseCruise), // سِیر: خارج
	}
	// هر دو segment افقی تداخل دارند، ولی فقط climb عمودی
	ctx := ctxRoute(RouteSourceWaypoints, ConfHigh, segs, map[int][]int{5: {0, 1}})
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", ctx)
	if r.Geo.ContextResult != CtxFullIntersection {
		t.Fatalf("صعود باید تداخل را بگیرد حتی وقتی سِیر نمی‌گیرد: %s", r.Geo.ContextResult)
	}
	// تأیید اینکه با ارتفاع سِیر ثابت (۳۵۰۰۰) از دست می‌رفت
	gcMiss := fullCtx(true, 5, 35000)
	if EvaluateImpact(n, model.RoleEnroute, "OIIX", gcMiss).Geo.ContextResult == CtxFullIntersection {
		t.Error("با ارتفاع سِیر ثابت نباید FULL می‌شد (این همان بهبود است)")
	}
}

// ۶) چند segment متقاطع با یک NOTAM
func TestRoute_MultipleSegmentsIntersect(t *testing.T) {
	n := airNotam(6, 0, 40000)
	segs := []SegmentBand{
		seg(1, 2, 20000, 30000, model.PhaseClimb),
		seg(2, 3, 30000, 35000, model.PhaseCruise),
		seg(3, 4, 35000, 35000, model.PhaseCruise),
	}
	ctx := ctxRoute(RouteSourceWaypoints, ConfHigh, segs, map[int][]int{6: {0, 1, 2}})
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", ctx)
	if len(r.Geo.MatchedSegments) != 3 {
		t.Errorf("انتظار ۳ segment متقاطع، دریافت %d", len(r.Geo.MatchedSegments))
	}
}

// ۷) waypoint با ترتیب اشتباه → مرتب‌سازی بر اساس sequence
func TestRoute_WaypointsWrongOrder(t *testing.T) {
	wps := model.Waypoints{
		{Sequence: 3, Lat: 40, Lon: 30},
		{Sequence: 1, Lat: 35, Lon: 51},
		{Sequence: 2, Lat: 38, Lon: 40},
	}
	legs := legsFromWaypoints(wps)
	if len(legs) != 2 {
		t.Fatalf("انتظار ۲ leg، دریافت %d", len(legs))
	}
	if legs[0].fromSeq != 1 || legs[0].toSeq != 2 || legs[1].fromSeq != 2 || legs[1].toSeq != 3 {
		t.Errorf("legها باید بر اساس sequence مرتب شوند: %+v", legs)
	}
}

// ۸) waypoint با مختصات نامعتبر → فیلتر می‌شود
func TestRoute_InvalidWaypointFiltered(t *testing.T) {
	wps := model.Waypoints{
		{Sequence: 1, Lat: 35, Lon: 51},
		{Sequence: 2, Lat: 200, Lon: 51}, // نامعتبر (lat خارج از بازه)
		{Sequence: 3, Lat: 40, Lon: 30},
	}
	legs := legsFromWaypoints(wps)
	// نقطهٔ نامعتبر حذف → ۲ نقطهٔ معتبر → ۱ leg
	if len(legs) != 1 || legs[0].fromSeq != 1 || legs[0].toSeq != 3 {
		t.Errorf("نقطهٔ نامعتبر باید حذف شود: %+v", legs)
	}
	// اگر کمتر از ۲ معتبر بماند → nil (fallback)
	if legsFromWaypoints(model.Waypoints{{Sequence: 1, Lat: 999, Lon: 0}}) != nil {
		t.Error("کمتر از ۲ نقطهٔ معتبر باید nil (fallback) بدهد")
	}
}

// ۹) profile ناقص برای بعضی segmentها → آن‌ها به cruise fallback می‌کنند
func TestRoute_PartialProfileFallsBackToCruise(t *testing.T) {
	fp := model.FlightPlan{
		CruiseAltitudeFt: 35000,
		RouteAltitudeProfile: model.AltProfile{
			{FromSequence: 1, ToSequence: 2, LowerFt: 5000, UpperFt: 18000, Phase: model.PhaseClimb},
			// segment 2→3 پوشش داده نشده
		},
	}
	climb := segmentBand(1, 2, fp)
	if climb.AltSource != AltSourceSegmentProfile || climb.Phase != model.PhaseClimb {
		t.Errorf("segment پوشش‌داده باید از پروفایل باشد: %+v", climb)
	}
	cruise := segmentBand(2, 3, fp)
	if cruise.AltSource != AltSourceCruiseFixed || cruise.LowerFt != 35000 {
		t.Errorf("segment بدون پروفایل باید به cruise fallback کند: %+v", cruise)
	}
}

// ۱۰) نبود waypoint → استفاده از fallback (تک‌segment، source=GREAT_CIRCLE)
func TestRoute_NoWaypointUsesFallback(t *testing.T) {
	if legsFromWaypoints(nil) != nil {
		t.Error("بدون waypoint باید nil بدهد تا service به GC برود")
	}
	// در fullCtx، source=GREAT_CIRCLE و confidence=MEDIUM
	ctx := fullCtx(true, 99, 35000)
	if ctx.Route.Source != RouteSourceGreatCircle || ctx.Route.Confidence != ConfMedium {
		t.Errorf("fallback باید GREAT_CIRCLE/MEDIUM باشد: %+v", ctx.Route)
	}
}

// ۱۱) نبود profile ولی وجود cruiseAltitudeFt
func TestRoute_NoProfileUsesCruise(t *testing.T) {
	fp := model.FlightPlan{CruiseAltitudeFt: 28000}
	b := segmentBand(0, 1, fp)
	if b.AltSource != AltSourceCruiseFixed || !b.AltKnown || b.LowerFt != 28000 {
		t.Errorf("بدون پروفایل باید cruise استفاده شود: %+v", b)
	}
}

// ۱۲) نبود هرگونه ارتفاع → عدم تشدید (segment altKnown=false → UNKNOWN_FLIGHT_LEVEL)
func TestRoute_NoAltitudeDoesNotEscalate(t *testing.T) {
	fp := model.FlightPlan{} // نه پروفایل، نه cruise
	b := segmentBand(0, 1, fp)
	if b.AltKnown || b.AltSource != AltSourceNone {
		t.Errorf("بدون هیچ ارتفاعی باید AltKnown=false باشد: %+v", b)
	}
	n := airNotam(12, 0, 40000)
	segs := []SegmentBand{b}
	ctx := ctxRoute(RouteSourceWaypoints, ConfHigh, segs, map[int][]int{12: {0}})
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", ctx)
	if r.Geo.ContextResult != CtxUnknownFlightLevel {
		t.Fatalf("انتظار UNKNOWN_FLIGHT_LEVEL، دریافت %s", r.Geo.ContextResult)
	}
	if r.Level == analysis.LevelCritical || r.Level == analysis.LevelHigh {
		t.Errorf("نبود ارتفاع نباید HIGH/CRITICAL شود (%s %d)", r.Level, r.Score)
	}
}

// ۱۳) مسیر مماس با محدوده (PostGIS مماس را تداخل می‌شمارد → segment در NotamSegments هست)
func TestRoute_Tangent(t *testing.T) {
	n := airNotam(13, 10000, 40000)
	segs := []SegmentBand{seg(1, 2, 34000, 34000, model.PhaseCruise)}
	ctx := ctxRoute(RouteSourceWaypoints, ConfHigh, segs, map[int][]int{13: {0}})
	if EvaluateImpact(n, model.RoleEnroute, "OIIX", ctx).Geo.ContextResult != CtxFullIntersection {
		t.Error("مماس با ارتفاعِ داخلِ بازه باید FULL_INTERSECTION شود")
	}
}

// ۱۴) NOTAM فعال فقط در بخشی از زمان پرواز → همچنان تداخل زمانی دارد (هم‌پوشانی)
func TestRoute_PartialTimeOverlap(t *testing.T) {
	from, to := win()
	n := airNotam(14, 0, 40000)
	// NOTAM از وسط پنجرهٔ پرواز شروع می‌شود و بعد از آن ادامه دارد → هم‌پوشانی جزئی
	n.EffectiveStart = from.Add(2 * time.Hour)
	end := to.Add(3 * time.Hour)
	n.EffectiveEnd = &end
	segs := []SegmentBand{seg(1, 2, 34000, 34000, model.PhaseCruise)}
	ctx := ctxRoute(RouteSourceWaypoints, ConfHigh, segs, map[int][]int{14: {0}})
	r := EvaluateImpact(n, model.RoleEnroute, "OIIX", ctx)
	if !r.Geo.TemporalIntersection {
		t.Error("هم‌پوشانی زمانی جزئی باید temporalIntersection=true بدهد")
	}
	if r.Geo.ContextResult != CtxFullIntersection {
		t.Errorf("با هم‌پوشانی زمانی و افقی/عمودی باید FULL شود، دریافت %s", r.Geo.ContextResult)
	}
}
