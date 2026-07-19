// Command calibration گزارش کالیبراسیون امتیازدهی را از دادهٔ واقعی تولید می‌کند (E3-calibration).
//
// خروجی‌ها:
//   - سطح A (Corpus): توزیع Base Potential برای نسخهٔ قبلی (1.1.0) و جدید (1.2.0) روی همهٔ NOTAMها.
//   - سطح B (Contextual): چند سناریوی پرواز واقعی و توزیع امتیاز نهایی.
//   - ۳۰ نمونهٔ مرزی برای کارشناس (JSON).
//
// اجرا: POSTGRES_HOST=localhost POSTGRES_PORT=5434 POSTGRES_PASSWORD=... go run ./cmd/calibration
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/hossein-repo/BaseProject/config"
	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"github.com/hossein-repo/BaseProject/internal/briefing"
	"github.com/hossein-repo/BaseProject/internal/messaging"
	"github.com/hossein-repo/BaseProject/internal/pipeline/analysis"
)

func main() {
	cfg := config.GetConfig()
	if err := db.InitDb(cfg); err != nil {
		panic(err)
	}
	defer db.CloseDb()

	var notams []model.Notam
	if err := db.GetDb().Where("raw_body <> ''").Find(&notams).Error; err != nil {
		panic(err)
	}
	fmt.Printf("\n══════════ گزارش کالیبراسیون امتیازدهی ══════════\n")
	fmt.Printf("مبنا: %d NOTAM واقعی | نسخهٔ فعال: %s\n", len(notams), analysis.Current.Version)

	corpusReport(notams)
	scenarioReport()
	writeSamples(notams)
}

// ---------------- سطح A: Corpus (Base Potential، بدون FlightPlan) ----------------

func corpusReport(notams []model.Notam) {
	v11 := analysis.ConfigByVersion("1.1.0")
	v12 := analysis.ConfigByVersion("1.2.0")

	lv11, lv12 := map[string]int{}, map[string]int{}
	status := map[string]int{}
	conf := map[string]int{}
	catLevel := map[string]map[string]int{} // category → level(v1.2) → count

	for _, n := range notams {
		ev, err := messaging.ParseNotamXML(n.RawBody)
		if err != nil {
			continue
		}
		ev.HumanReadableText = messaging.EnsureHumanReadableText(ev)
		r11 := analysis.AnalyzeWith(ev, v11)
		r12 := analysis.AnalyzeWith(ev, v12)
		lv11[r11.BaseLevel]++
		lv12[r12.BaseLevel]++
		status[r12.CorpusStatus]++
		conf[r12.Confidence]++
		if catLevel[r12.Category] == nil {
			catLevel[r12.Category] = map[string]int{}
		}
		catLevel[r12.Category][r12.BaseLevel]++
	}

	total := len(notams)
	fmt.Printf("\n─── سطح A: Base Potential (بدون FlightPlan) ───\n")
	fmt.Printf("%-10s %12s %12s\n", "سطح", "v1.1.0", "v1.2.0")
	for _, lv := range levels() {
		fmt.Printf("%-10s %12s %12s\n", lv, pct(lv11[lv], total), pct(lv12[lv], total))
	}
	fmt.Printf("\nوضعیت Corpus (v1.2.0):\n")
	for _, s := range []string{analysis.CorpusInformationalOnly, analysis.CorpusContextRequired, analysis.CorpusBaseOnly} {
		fmt.Printf("  %-20s %s\n", s, pct(status[s], total))
	}
	fmt.Printf("\nاطمینان تحلیل پایه (v1.2.0):\n")
	for _, c := range []string{"HIGH", "MEDIUM", "LOW"} {
		fmt.Printf("  %-8s %s\n", c, pct(conf[c], total))
	}
	fmt.Printf("\n⚠️ سطح A = «سقف بالقوه»؛ CONTEXT_REQUIRED یعنی امتیاز نهایی نیازمند بستر پرواز است.\n")
}

// ---------------- سطح B: سناریوهای پرواز واقعی ----------------

func scenarioReport() {
	svc := briefing.NewService()
	base := time.Date(2022, 2, 2, 8, 0, 0, 0, time.UTC)
	end := base.Add(6 * time.Hour)
	fp := func(m model.FlightPlan) model.FlightPlan {
		m.ETD, m.ETA, m.BufferMinutes = base, end, 180
		return m
	}
	firs := model.StringSlice{"LRBB", "LHCC", "LOVV", "LBSR", "LTBB"}

	scenarios := []struct {
		name string
		fp   model.FlightPlan
	}{
		{"۱ فرودگاه تک‌باند (LIRN)", fp(model.FlightPlan{ADEP: "LFBD", ADES: "LIRN", AircraftCategory: "JET", FlightRules: "IFR", CruiseAltitudeFt: 35000})},
		{"۲ فرودگاه چندباند (LTFM ۵باند)", fp(model.FlightPlan{ADEP: "LFBD", ADES: "LTFM", AircraftCategory: "JET", FlightRules: "IFR", CruiseAltitudeFt: 35000})},
		{"۳ پرواز IFR", fp(model.FlightPlan{ADEP: "EDDF", ADES: "LTFM", EnrouteFIRs: firs, FlightRules: "IFR", AircraftCategory: "JET", CruiseAltitudeFt: 35000})},
		{"۴ پرواز VFR", fp(model.FlightPlan{ADEP: "EDDF", ADES: "LTFM", EnrouteFIRs: firs, FlightRules: "VFR", AircraftCategory: "PISTON", CruiseAltitudeFt: 8000})},
		{"۵ جت (AVGAS بی‌ربط)", fp(model.FlightPlan{ADEP: "EIDW", ADES: "LERS", AircraftCategory: "JET", FlightRules: "IFR", CruiseAltitudeFt: 35000})},
		{"۶ مسیر waypoint واقعی (انحراف)", fp(model.FlightPlan{ADEP: "EDDF", ADES: "LTFM", EnrouteFIRs: firs, AircraftCategory: "JET", FlightRules: "IFR", CruiseAltitudeFt: 20000, RouteWaypoints: model.Waypoints{{Sequence: 1, Lat: 50.03, Lon: 8.57}, {Sequence: 2, Lat: 44, Lon: 18}, {Sequence: 3, Lat: 40, Lon: 24}, {Sequence: 4, Lat: 41.26, Lon: 28.74}}})},
		{"۷ تداخل فقط climb (پروفایل)", fp(model.FlightPlan{ADEP: "LRCL", ADES: "LRBS", EnrouteFIRs: firs, AircraftCategory: "JET", FlightRules: "IFR", CruiseAltitudeFt: 35000, RouteWaypoints: model.Waypoints{{Sequence: 1, Lat: 46.78, Lon: 23.68}, {Sequence: 2, Lat: 44.5, Lon: 26.08}}, RouteAltitudeProfile: model.AltProfile{{FromSequence: 1, ToSequence: 2, LowerFt: 0, UpperFt: 18000, Phase: "CLIMB"}}})},
		{"۸ Airspace داخل مسیر FL200", fp(model.FlightPlan{ADEP: "EDDF", ADES: "LTFM", EnrouteFIRs: firs, AircraftCategory: "JET", FlightRules: "IFR", CruiseAltitudeFt: 20000})},
		{"۹ Airspace بالای ارتفاع FL350", fp(model.FlightPlan{ADEP: "EDDF", ADES: "LTFM", EnrouteFIRs: firs, AircraftCategory: "JET", FlightRules: "IFR", CruiseAltitudeFt: 35000})},
		{"۱۰ با الترنت", fp(model.FlightPlan{ADEP: "EIDW", ADES: "LERS", Alternates: model.StringSlice{"LEBL"}, AircraftCategory: "JET", FlightRules: "IFR", CruiseAltitudeFt: 35000})},
		{"۱۱ بدون ارتفاع (UNKNOWN_FL)", fp(model.FlightPlan{ADEP: "EDDF", ADES: "LTFM", EnrouteFIRs: firs, AircraftCategory: "JET", FlightRules: "IFR"})},
	}

	fmt.Printf("\n─── سطح B: سناریوهای پرواز واقعی (امتیاز نهاییِ Contextual) ───\n")
	agg := map[string]int{}
	ctxRes := map[string]int{}
	for _, sc := range scenarios {
		b, err := svc.Build(sc.fp)
		if err != nil {
			fmt.Printf("  %s → خطا: %v\n", sc.name, err)
			continue
		}
		fmt.Printf("  %-34s کل=%-3d %v\n", sc.name, b.TotalCount, compactLevels(b.CountsByLevel))
		for lv, c := range b.CountsByLevel {
			agg[lv] += c
		}
		for _, g := range b.Groups {
			for _, it := range g.Items {
				if it.Geo != nil {
					ctxRes[it.Geo.ContextResult]++
				}
			}
		}
	}
	fmt.Printf("\nتجمیع سطوح نهایی همهٔ سناریوها: %v\n", agg)
	fmt.Printf("توزیع contextResult فضای هوایی: %v\n", ctxRes)
	fmt.Printf("\n⚠️ سطح B با دادهٔ واقعی تولید شد (fixture نیست). هدف ~۶٪ HIGH فقط معیار بررسی است، تحمیل نشده.\n")
}

// ---------------- ۳۰ نمونهٔ مرزی برای کارشناس ----------------

type sample struct {
	NotamID           int      `json:"notamId"`
	Bucket            string   `json:"bucket"`
	RawText           string   `json:"rawText"`
	QCode             string   `json:"qCode"`
	Category          string   `json:"category"`
	OperationalEffect string   `json:"operationalEffect"`
	BasePotentialV11  int      `json:"basePotentialV11"`
	BasePotential     int      `json:"basePotential"`
	FinalScore        int      `json:"finalScore,omitempty"`
	Priority          string   `json:"priority"`
	ContextResult     string   `json:"contextResult,omitempty"`
	RecommendedAction string   `json:"recommendedAction,omitempty"`
	Confidence        string   `json:"confidence"`
	CorpusStatus      string   `json:"corpusStatus,omitempty"`
	Reasons           []string `json:"reasons,omitempty"`
	MissingData       []string `json:"missingData,omitempty"`
	ScoringVersion    string   `json:"scoringVersion"`
}

func writeSamples(notams []model.Notam) {
	v11 := analysis.ConfigByVersion("1.1.0")
	v12 := analysis.ConfigByVersion("1.2.0")

	type scored struct {
		n        model.Notam
		r11, r12 analysis.Result
	}
	var all []scored
	for _, n := range notams {
		ev, err := messaging.ParseNotamXML(n.RawBody)
		if err != nil {
			continue
		}
		ev.HumanReadableText = messaging.EnsureHumanReadableText(ev)
		all = append(all, scored{n, analysis.AnalyzeWith(ev, v11), analysis.AnalyzeWith(ev, v12)})
	}

	pick := func(list []scored, n int, bucket string) []sample {
		var out []sample
		for i := 0; i < len(list) && i < n; i++ {
			s := list[i]
			out = append(out, sample{
				NotamID: s.n.Id, Bucket: bucket, RawText: trim(s.n.PlainText, 160),
				QCode: s.r12.QCode, Category: s.r12.Category,
				BasePotentialV11: s.r11.BaseScore, BasePotential: s.r12.BaseScore,
				Priority: s.r12.BaseLevel, Confidence: s.r12.Confidence,
				CorpusStatus: s.r12.CorpusStatus, ScoringVersion: s.r12.ScoringVersion,
			})
		}
		return out
	}

	var out []sample
	// ۵ افزایش پایه (v1.1.0 → v1.2.0)
	sort.Slice(all, func(i, j int) bool {
		return (all[i].r12.BaseScore - all[i].r11.BaseScore) > (all[j].r12.BaseScore - all[j].r11.BaseScore)
	})
	out = append(out, pick(all, 5, "BASE_INCREASE")...)
	// ۵ نزدیک مرز MEDIUM/HIGH (۶۰)
	sort.Slice(all, func(i, j int) bool { return absi(all[i].r12.BaseScore-60) < absi(all[j].r12.BaseScore-60) })
	out = append(out, pick(all, 5, "NEAR_MED_HIGH")...)
	// ۵ نزدیک مرز LOW/MEDIUM (۳۵)
	sort.Slice(all, func(i, j int) bool { return absi(all[i].r12.BaseScore-35) < absi(all[j].r12.BaseScore-35) })
	out = append(out, pick(all, 5, "NEAR_LOW_MED")...)
	// ۵ اطمینان پایین
	var low []scored
	for _, s := range all {
		if s.r12.Confidence == "LOW" {
			low = append(low, s)
		}
	}
	out = append(out, pick(low, 5, "LOW_CONFIDENCE")...)

	// ۱۰ کاهش شدید در بستر پرواز واقعی (contextual)
	out = append(out, contextDecreases(10)...)

	f := "docs/phase1/calibration_samples.json"
	b, _ := json.MarshalIndent(out, "", "  ")
	_ = os.WriteFile("../"+f, b, 0644)
	fmt.Printf("\n─── ۳۰ نمونهٔ مرزی برای کارشناس ───\n")
	fmt.Printf("%d نمونه در %s نوشته شد (buckets: BASE_INCREASE/NEAR_MED_HIGH/NEAR_LOW_MED/LOW_CONFIDENCE/CONTEXT_DECREASE)\n", len(out), f)
}

// contextDecreases NOTAMهایی که در یک پرواز واقعی، امتیاز نهاییِ کانتکستی‌شان به‌شدت
// از Base Potential کمتر شده (مثلاً Airspace خارج از مسیر، سوخت بی‌ربط).
func contextDecreases(limit int) []sample {
	svc := briefing.NewService()
	fp := model.FlightPlan{
		ADEP: "EDDF", ADES: "LTFM",
		EnrouteFIRs:      model.StringSlice{"LRBB", "LHCC", "LOVV", "LBSR", "LTBB"},
		AircraftCategory: "JET", FlightRules: "IFR", CruiseAltitudeFt: 35000,
		ETD: time.Date(2022, 2, 2, 8, 0, 0, 0, time.UTC),
		ETA: time.Date(2022, 2, 2, 14, 0, 0, 0, time.UTC), BufferMinutes: 180,
	}
	b, err := svc.Build(fp)
	if err != nil {
		return nil
	}
	type item struct {
		it   briefing.Item
		drop int
	}
	var items []item
	for _, g := range b.Groups {
		for _, it := range g.Items {
			drop := it.Notam.BaseScore - it.ContextualScore
			if drop > 20 {
				items = append(items, item{it, drop})
			}
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].drop > items[j].drop })
	var out []sample
	for i := 0; i < len(items) && i < limit; i++ {
		it := items[i].it
		s := sample{
			NotamID: it.Notam.ID, Bucket: "CONTEXT_DECREASE", RawText: trim(it.Notam.PlainText, 160),
			QCode: it.Notam.QCode, Category: it.Notam.Category, OperationalEffect: it.Effect,
			BasePotential: it.Notam.BaseScore, FinalScore: it.ContextualScore, Priority: it.ContextualLevel,
			RecommendedAction: it.Action, ScoringVersion: analysis.Current.Version,
		}
		if it.Geo != nil {
			s.ContextResult = it.Geo.ContextResult
			s.Confidence = it.Geo.Confidence
			s.Reasons = it.Geo.Reasons
			s.MissingData = it.Geo.MissingData
		}
		out = append(out, s)
	}
	return out
}

// ---------------- helpers ----------------

func levels() []string {
	return []string{analysis.LevelCritical, analysis.LevelHigh, analysis.LevelMedium, analysis.LevelLow, analysis.LevelInfo}
}
func pct(n, total int) string {
	if total == 0 {
		return "0"
	}
	return fmt.Sprintf("%d (%.0f%%)", n, 100*float64(n)/float64(total))
}
func compactLevels(m map[string]int) string {
	s := ""
	for _, lv := range levels() {
		if m[lv] > 0 {
			s += fmt.Sprintf("%s=%d ", lv[:1], m[lv])
		}
	}
	return s
}
func absi(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
func trim(s string, n int) string {
	s = normalizeSpace(s)
	if len(s) > n {
		return s[:n]
	}
	return s
}
func normalizeSpace(s string) string {
	out := make([]rune, 0, len(s))
	for _, r := range s {
		if r == '\n' || r == '\r' || r == '\t' {
			r = ' '
		}
		out = append(out, r)
	}
	return string(out)
}
