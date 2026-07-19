package briefing

import (
	"testing"

	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/pipeline/analysis"
	"github.com/hossein-repo/BaseProject/internal/pipeline/qcode"
)

func rwyClosed(loc string, base int) model.Notam {
	return model.Notam{
		LocationICAO: loc, Category: qcode.CatRunway, QCondition: "LC",
		Tags: model.StringSlice{analysis.TagRwyClosed}, PlainText: "RWY 10/28 CLSD",
		BaseScore: base, BaseLevel: analysis.LevelFor(base),
	}
}

// فاکتور تعداد باند (مثال شاخص کارشناس): بستن باند در فرودگاه تک‌باند بحرانی‌تر از چندباند است.
func TestImpact_RunwayCount(t *testing.T) {
	n := rwyClosed("LERS", 90)

	single := EvaluateImpact(n, model.RoleADES, "LERS", FlightContext{RunwayCounts: map[string]int{"LERS": 1}})
	multi := EvaluateImpact(n, model.RoleADES, "LERS", FlightContext{RunwayCounts: map[string]int{"LERS": 4}})

	if single.Score <= multi.Score {
		t.Errorf("باند بسته در تک‌باند (%d) باید > چندباند (%d)", single.Score, multi.Score)
	}
	if single.Effect != EffectAerodromeUnusable {
		t.Errorf("تک‌باند بسته باید اثر AERODROME_UNUSABLE بگیرد، دریافت %s", single.Effect)
	}
	if multi.Score >= 80 {
		t.Errorf("یک باند از چهار نباید بحرانی بماند (%d)", multi.Score)
	}
}

// «بسته به VFR» برای پرواز IFR بی‌اثر است.
func TestImpact_FlightRulesNotApplicable(t *testing.T) {
	closedToVFR := model.Notam{
		LocationICAO: "OIII", Category: qcode.CatRunway, QCondition: "LV",
		PlainText: "RWY 11 CLOSED TO VFR", BaseScore: 60, BaseLevel: analysis.LevelHigh,
	}
	ifr := EvaluateImpact(closedToVFR, model.RoleADES, "OIII", FlightContext{FlightRules: model.RulesIFR})
	if ifr.Effect != EffectNotApplicable {
		t.Errorf("«بسته به VFR» برای پرواز IFR باید NOT_APPLICABLE شود، دریافت %s", ifr.Effect)
	}
	if ifr.Score >= 35 {
		t.Errorf("مورد بی‌ربط باید امتیاز پایین بگیرد (%d)", ifr.Score)
	}
	// همان NOTAM برای پرواز VFR باید مرتبط بماند
	vfr := EvaluateImpact(closedToVFR, model.RoleADES, "OIII", FlightContext{FlightRules: model.RulesVFR})
	if vfr.Effect == EffectNotApplicable {
		t.Error("برای پرواز VFR نباید بی‌ربط شود")
	}
}

// سوخت 100LL برای هواپیمای جت بی‌ربط است (مثال مستقیم کارشناس).
func TestImpact_FuelRelevance(t *testing.T) {
	avgas := model.Notam{
		LocationICAO: "OIII", Category: qcode.CatService,
		PlainText: "100LL NOT AVBL", BaseScore: 50, BaseLevel: analysis.LevelMedium,
	}
	jet := EvaluateImpact(avgas, model.RoleADEP, "OIII", FlightContext{AircraftCategory: model.AircraftJet})
	if jet.Effect != EffectNotApplicable {
		t.Errorf("AVGAS برای جت باید NOT_APPLICABLE شود، دریافت %s", jet.Effect)
	}
	// همان برای هواپیمای پیستونی مرتبط است
	piston := EvaluateImpact(avgas, model.RoleADEP, "OIII", FlightContext{AircraftCategory: model.AircraftPiston})
	if piston.Effect == EffectNotApplicable {
		t.Error("AVGAS برای هواپیمای پیستونی نباید بی‌ربط شود")
	}
	// سوخت جت برای جت مرتبط است
	jetFuel := model.Notam{LocationICAO: "OIII", Category: qcode.CatService, PlainText: "JET A1 NOT AVBL", BaseScore: 50}
	if EvaluateImpact(jetFuel, model.RoleADEP, "OIII", FlightContext{AircraftCategory: model.AircraftJet}).Effect == EffectNotApplicable {
		t.Error("سوخت جت برای هواپیمای جت باید مرتبط بماند")
	}
}

// تست #16: CRITICAL نهایی فقط با context تأییدشده صادر می‌شود، نه صرفاً Base Potential بالا.
func TestCriticalRequiresConfirmedContext(t *testing.T) {
	// باند بسته با Base Potential 100، ولی در فرودگاه چندباند → نباید CRITICAL بماند
	rwy := model.Notam{
		LocationICAO: "LTFM", Category: qcode.CatRunway, QCondition: "LC",
		Tags: model.StringSlice{analysis.TagRwyClosed}, PlainText: "RWY 18 CLSD", BaseScore: 100,
	}
	multi := EvaluateImpact(rwy, model.RoleADES, "LTFM", FlightContext{RunwayCounts: map[string]int{"LTFM": 5}})
	if multi.Level == analysis.LevelCritical {
		t.Errorf("باند بسته در فرودگاه ۵باند نباید CRITICAL بماند (%d)", multi.Score)
	}
	// همان با تک‌باند → context تأییدشده → CRITICAL مجاز
	single := EvaluateImpact(rwy, model.RoleADES, "LTFM", FlightContext{RunwayCounts: map[string]int{"LTFM": 1}})
	if single.Level != analysis.LevelCritical {
		t.Errorf("باند بسته در تک‌باند باید CRITICAL شود (%d)", single.Score)
	}
	// airspace با Base Potential بالا ولی خارج از مسیر → نباید CRITICAL شود
	air := model.Notam{
		Category: qcode.CatRestriction, BaseScore: 90, AreaRadiusNM: 20, VerticalKnown: true, UpperFt: 40000,
	}
	off := EvaluateImpact(air, model.RoleEnroute, "OIIX", FlightContext{
		Route: RouteContext{Source: RouteSourceWaypoints, Confidence: ConfHigh,
			Segments: []SegmentBand{seg2(0, 1, 35000, 35000)}, NotamSegments: map[int][]int{}},
	})
	if off.Level == analysis.LevelCritical || off.Level == analysis.LevelHigh {
		t.Errorf("airspace خارج از مسیر نباید CRITICAL/HIGH بماند (%s %d)", off.Level, off.Score)
	}
}

func seg2(from, to, lower, upper int) SegmentBand {
	return SegmentBand{FromSeq: from, ToSeq: to, LowerFt: lower, UpperFt: upper, AltKnown: true, AltSource: AltSourceCruiseFixed, Phase: model.PhaseCruise}
}

func TestImpact_OperationalEffectMapping(t *testing.T) {
	cases := []struct {
		n    model.Notam
		want string
	}{
		{model.Notam{Category: qcode.CatAerodrome, Tags: model.StringSlice{analysis.TagAdClosed}}, EffectAerodromeUnusable},
		{model.Notam{Category: qcode.CatILS}, EffectApproachDegraded},
		{model.Notam{Category: qcode.CatNavigation}, EffectNavDegradation},
		{model.Notam{Category: qcode.CatRestriction}, EffectRouteRestriction},
		{model.Notam{Category: qcode.CatObstacle}, EffectObstacleHazard},
		{model.Notam{Category: qcode.CatOther, QCondition: "TT"}, EffectInformationalOnly},
	}
	for _, c := range cases {
		if got := operationalEffect(c.n); got != c.want {
			t.Errorf("effect=%s، انتظار %s (category=%s)", got, c.want, c.n.Category)
		}
	}
}

// هیچ NOTAMی بی‌صاحب نماند: مورد بی‌ربط هم امتیاز پایین می‌گیرد، نه صفر/حذف.
func TestImpact_NothingOrphaned(t *testing.T) {
	avgas := model.Notam{Category: qcode.CatService, PlainText: "100LL NOT AVBL", BaseScore: 50}
	r := EvaluateImpact(avgas, model.RoleADEP, "OIII", FlightContext{AircraftCategory: model.AircraftJet})
	if r.Score <= 0 {
		t.Error("مورد بی‌ربط باید امتیاز مثبتِ پایین بگیرد، نه صفر")
	}
	if r.Action == "" {
		t.Error("هر آیتم باید اقدام پیشنهادی داشته باشد")
	}
}
