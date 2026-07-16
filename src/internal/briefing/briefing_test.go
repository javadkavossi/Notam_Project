package briefing

import (
	"testing"
	"time"

	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/pipeline/analysis"
	"github.com/hossein-repo/BaseProject/internal/pipeline/qcode"
)

func testFlight() model.FlightPlan {
	etd := time.Date(2026, 7, 20, 6, 0, 0, 0, time.UTC)
	return model.FlightPlan{
		BaseModel:     model.BaseModel{Id: 1},
		Username:      "pilot",
		ADEP:          "OIII",
		ADES:          "OIIE",
		Alternates:    model.StringSlice{"OISS"},
		EnrouteFIRs:   model.StringSlice{"OIIX"},
		ETD:           etd,
		ETA:           etd.Add(2 * time.Hour),
		BufferMinutes: 60,
	}
}

func notam(id int, loc, category string, base int) model.Notam {
	return model.Notam{
		BaseModel:      model.BaseModel{Id: id},
		LocationICAO:   loc,
		Category:       category,
		BaseScore:      base,
		BaseLevel:      analysis.LevelFor(base),
		EffectiveStart: time.Date(2026, 7, 20, 5, 0, 0, 0, time.UTC),
	}
}

func TestClassifyRole(t *testing.T) {
	fp := testFlight()
	cases := []struct {
		loc      string
		fir      string
		wantRole string
	}{
		{"OIIE", "", model.RoleADES},
		{"OIII", "", model.RoleADEP},
		{"OISS", "", model.RoleALTN},
		{"ZZZZ", "OIIX", model.RoleEnroute},
	}
	for _, c := range cases {
		n := model.Notam{LocationICAO: c.loc, AffectedFIR: c.fir}
		role, _ := classifyRole(fp, n)
		if role != c.wantRole {
			t.Errorf("loc=%s fir=%s → role=%s، انتظار %s", c.loc, c.fir, role, c.wantRole)
		}
	}
}

func TestClassifyRole_CaseInsensitive(t *testing.T) {
	fp := testFlight()
	role, icao := classifyRole(fp, model.Notam{LocationICAO: "oiie"})
	if role != model.RoleADES || icao != "OIIE" {
		t.Errorf("تطابق باید بدون حساسیت به حروف باشد: role=%s icao=%s", role, icao)
	}
}

// همان NOTAM باید در مقصد امتیاز بیشتری از الترنت بگیرد (اصل وابستگی به کانتکست پرواز).
func TestContextualScore_DependsOnRole(t *testing.T) {
	n := notam(1, "OIIE", qcode.CatRunway, 70)

	ades := contextualScore(n, model.RoleADES)
	altn := contextualScore(n, model.RoleALTN)
	enroute := contextualScore(n, model.RoleEnroute)

	if !(ades > altn) {
		t.Errorf("باند در مقصد (%d) باید > الترنت (%d)", ades, altn)
	}
	if !(altn > enroute) {
		t.Errorf("باند در الترنت (%d) باید > مسیر (%d)", altn, enroute)
	}
}

func TestContextualScore_ArrivalRelevance(t *testing.T) {
	// ILS در مقصد مرتبط با فرود است → بیشتر از یک دستهٔ نامرتبط با فرود (اپرون)
	ils := contextualScore(notam(1, "OIIE", qcode.CatILS, 50), model.RoleADES)
	apron := contextualScore(notam(2, "OIIE", qcode.CatApron, 50), model.RoleADES)
	if !(ils > apron) {
		t.Errorf("ILS مقصد (%d) باید > اپرون مقصد (%d)", ils, apron)
	}
}

func TestContextualScore_Clamped(t *testing.T) {
	if s := contextualScore(notam(1, "OIIE", qcode.CatRunway, 100), model.RoleADES); s != 100 {
		t.Errorf("امتیاز باید در ۱۰۰ کلَمپ شود، دریافت %d", s)
	}
}

func TestBuild_GroupsAndCritical(t *testing.T) {
	fp := testFlight()
	notams := []model.Notam{
		notam(1, "OIIE", qcode.CatRunway, 90),   // مقصد، بحرانی
		notam(2, "OIII", qcode.CatTaxiway, 40),  // مبدأ
		notam(3, "OISS", qcode.CatAerodrome, 30), // الترنت
		{BaseModel: model.BaseModel{Id: 4}, AffectedFIR: "OIIX", Category: qcode.CatAirspace, BaseScore: 50},
	}
	b := Build(fp, notams)

	if b.TotalCount != 4 {
		t.Errorf("TotalCount=%d، انتظار ۴", b.TotalCount)
	}
	// ترتیب گروه‌ها باید ثابت باشد: ADEP → ADES → ALTN → ENROUTE
	wantOrder := []string{model.RoleADEP, model.RoleADES, model.RoleALTN, model.RoleEnroute}
	if len(b.Groups) != 4 {
		t.Fatalf("انتظار ۴ گروه، دریافت %d", len(b.Groups))
	}
	for i, g := range b.Groups {
		if g.Role != wantOrder[i] {
			t.Errorf("گروه %d = %s، انتظار %s", i, g.Role, wantOrder[i])
		}
	}
	// NOTAM باند مقصد باید در خلاصهٔ بحرانی باشد
	if len(b.Critical) == 0 {
		t.Fatal("خلاصهٔ بحرانی نباید خالی باشد")
	}
	if b.Critical[0].Notam.ID != 1 {
		t.Errorf("مورد بحرانی اول باید NOTAM #1 باشد، دریافت #%d", b.Critical[0].Notam.ID)
	}
	if b.CountsByLevel[analysis.LevelCritical] < 1 {
		t.Errorf("شمارش سطح بحرانی نادرست: %+v", b.CountsByLevel)
	}
}

func TestBuild_SortsByContextualScoreDesc(t *testing.T) {
	fp := testFlight()
	notams := []model.Notam{
		notam(1, "OIIE", qcode.CatApron, 25),
		notam(2, "OIIE", qcode.CatRunway, 80),
		notam(3, "OIIE", qcode.CatLighting, 45),
	}
	b := Build(fp, notams)
	var ades *Group
	for i := range b.Groups {
		if b.Groups[i].Role == model.RoleADES {
			ades = &b.Groups[i]
		}
	}
	if ades == nil {
		t.Fatal("گروه مقصد یافت نشد")
	}
	for i := 1; i < len(ades.Items); i++ {
		if ades.Items[i-1].ContextualScore < ades.Items[i].ContextualScore {
			t.Errorf("مرتب‌سازی نزولی نیست: %d سپس %d",
				ades.Items[i-1].ContextualScore, ades.Items[i].ContextualScore)
		}
	}
}

func TestBuild_WindowAndMatchReason(t *testing.T) {
	fp := testFlight()
	b := Build(fp, []model.Notam{notam(1, "OIIE", qcode.CatRunway, 70)})

	from, to := fp.Window()
	if !b.WindowFrom.Equal(from) || !b.WindowTo.Equal(to) {
		t.Errorf("پنجرهٔ زمانی نادرست: %v..%v", b.WindowFrom, b.WindowTo)
	}
	item := b.Groups[0].Items[0]
	if item.MatchReason == "" {
		t.Error("match_reason نباید خالی باشد")
	}
}

func TestFlightPlanWindowAndAirports(t *testing.T) {
	fp := testFlight()
	from, to := fp.Window()
	if !from.Equal(fp.ETD.Add(-60*time.Minute)) || !to.Equal(fp.ETA.Add(60*time.Minute)) {
		t.Errorf("buffer اعمال نشد: %v..%v", from, to)
	}
	apts := fp.Airports()
	if len(apts) != 3 {
		t.Errorf("انتظار ۳ فرودگاه (مبدأ/مقصد/الترنت)، دریافت %v", apts)
	}
}
