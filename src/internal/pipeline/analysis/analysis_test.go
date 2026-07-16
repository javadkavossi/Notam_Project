package analysis

import (
	"testing"

	"github.com/hossein-repo/BaseProject/internal/messaging"
	"github.com/hossein-repo/BaseProject/internal/pipeline/qcode"
)

func hasTag(tags []string, t string) bool {
	for _, x := range tags {
		if x == t {
			return true
		}
	}
	return false
}

// مجموعهٔ آزمون طلایی (E3-6): NOTAMهای نماینده با نتیجهٔ موردانتظار.
func TestGoldenAnalyze(t *testing.T) {
	cases := []struct {
		name      string
		ev        messaging.NotamEvent
		wantCat   string
		wantLevel string
		wantTag   string // "" یعنی بررسی نشود
	}{
		{
			name:      "runway closed (QMRLC)",
			ev:        messaging.NotamEvent{QCode: "QMRLC", Text: "RWY 29R CLSD DUE WIP"},
			wantCat:   qcode.CatRunway,
			wantLevel: LevelCritical, // 70 + 30 closed + RWY_CLOSED tag
			wantTag:   TagRwyClosed,
		},
		{
			name:      "ILS unserviceable (QICAS)",
			ev:        messaging.NotamEvent{QCode: "QICAS", Text: "ILS RWY 29R U/S"},
			wantCat:   qcode.CatILS,
			wantLevel: LevelHigh, // 50 + 22 AS + 6 ILS_OUT = 78 → HIGH
			wantTag:   TagILSOut,
		},
		{
			name:      "aerodrome closed (QFALC)",
			ev:        messaging.NotamEvent{QCode: "QFALC", Text: "AD CLSD"},
			wantCat:   qcode.CatAerodrome,
			wantLevel: LevelCritical,
			wantTag:   TagAdClosed,
		},
		{
			name:      "taxiway closed (QMXLC)",
			ev:        messaging.NotamEvent{QCode: "QMXLC", Text: "TWY B CLSD"},
			wantCat:   qcode.CatTaxiway,
			wantLevel: LevelHigh, // 38 + 30 LC = 68 → HIGH
		},
		{
			name:      "cancelled notam (QMRCN)",
			ev:        messaging.NotamEvent{QCode: "QMRCN", Text: "A0044/26 CANCELLED"},
			wantCat:   qcode.CatRunway,
			wantLevel: LevelLow, // 70 - 50 CN = 20 → LOW
		},
		{
			name:      "restricted area active (QRTCA)",
			ev:        messaging.NotamEvent{QCode: "QRTCA", Text: "RESTRICTED AREA ACT"},
			wantCat:   qcode.CatRestriction,
			wantLevel: LevelHigh, // 55 + 12
		},
		{
			name:      "obstacle crane (QOBCE)",
			ev:        messaging.NotamEvent{QCode: "QOBCE", Text: "CRANE ERECTED"},
			wantCat:   qcode.CatObstacle,
			wantTag:   TagObstacle,
		},
		{
			name:      "FICON via text fallback (no Q-code)",
			ev:        messaging.NotamEvent{Text: "RWY 29 FICON 5/5/5 WET"},
			wantCat:   qcode.CatRunway,
			wantTag:   TagFICON,
		},
		{
			name:      "runway closed via text fallback",
			ev:        messaging.NotamEvent{Text: "RWY 04L/22R CLOSED"},
			wantCat:   qcode.CatRunway,
			wantLevel: LevelCritical,
			wantTag:   TagRwyClosed,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := Analyze(c.ev)
			if got.Category != c.wantCat {
				t.Errorf("category=%q، انتظار %q", got.Category, c.wantCat)
			}
			if c.wantLevel != "" && got.BaseLevel != c.wantLevel {
				t.Errorf("level=%q (score=%d)، انتظار %q", got.BaseLevel, got.BaseScore, c.wantLevel)
			}
			if c.wantTag != "" && !hasTag(got.Tags, c.wantTag) {
				t.Errorf("tag %q یافت نشد؛ tags=%v", c.wantTag, got.Tags)
			}
		})
	}
}

// حفاظ ایمنی (رگرسیون از دادهٔ واقعی): «rapid exit taxiway» نباید «باند» شود.
// قبلاً QMYLC به‌دلیل نبودِ MY در جدول، با fallback حرف اول به RUNWAY می‌رفت و با
// برچسب RWY_CLOSED امتیاز ۱۰۰/CRITICAL می‌گرفت — یعنی بستنِ تاکسی‌وی به‌عنوان
// بستنِ باند به خلبان گزارش می‌شد.
func TestRapidExitTaxiwayNotClassifiedAsRunway(t *testing.T) {
	got := Analyze(messaging.NotamEvent{QCode: "QMYLC", Text: "RAPID EXIT TWY N5 CLOSED"})

	if got.Category == qcode.CatRunway {
		t.Errorf("QMYLC (rapid exit taxiway) نباید RUNWAY شود")
	}
	if got.Category != qcode.CatTaxiway {
		t.Errorf("category=%q، انتظار TAXIWAY", got.Category)
	}
	if hasTag(got.Tags, TagRwyClosed) {
		t.Errorf("برچسب RWY_CLOSED نباید برای تاکسی‌وی صادر شود؛ tags=%v", got.Tags)
	}
	if got.BaseLevel == LevelCritical {
		t.Errorf("بستن تاکسی‌وی نباید بحرانی باشد (score=%d)", got.BaseScore)
	}
}

// موضوع ناشناختهٔ گروه M نباید به RUNWAY (پرخطرترین دستهٔ گروه) نگاشت شود و
// نباید برچسب قاطع بگیرد.
func TestUnknownSubjectDoesNotEscalate(t *testing.T) {
	got := Analyze(messaging.NotamEvent{QCode: "QMZLC", Text: "SOMETHING CLOSED"})

	if got.Category == qcode.CatRunway {
		t.Errorf("موضوع ناشناختهٔ MZ نباید RUNWAY شود")
	}
	if hasTag(got.Tags, TagRwyClosed) {
		t.Errorf("موضوع حدس‌زده‌شده نباید برچسب RWY_CLOSED بگیرد؛ tags=%v", got.Tags)
	}
	// باید همچنان از باندِ بسته کم‌اهمیت‌تر باشد
	rwy := Analyze(messaging.NotamEvent{QCode: "QMRLC", Text: "RWY CLSD"}).BaseScore
	if got.BaseScore >= rwy {
		t.Errorf("ناشناخته (%d) نباید ≥ باند بسته (%d) باشد", got.BaseScore, rwy)
	}
}

func TestScoreOrdering(t *testing.T) {
	// باند بسته باید مهم‌تر از تاکسی‌وی بسته باشد؛ هر دو از NOTAM لغوشده مهم‌تر
	rwy := Analyze(messaging.NotamEvent{QCode: "QMRLC", Text: "RWY CLSD"}).BaseScore
	twy := Analyze(messaging.NotamEvent{QCode: "QMXLC", Text: "TWY CLSD"}).BaseScore
	cancelled := Analyze(messaging.NotamEvent{QCode: "QMRCN", Text: "CANCELLED"}).BaseScore

	if !(rwy > twy) {
		t.Errorf("باند بسته (%d) باید > تاکسی‌وی بسته (%d)", rwy, twy)
	}
	if !(twy > cancelled) {
		t.Errorf("تاکسی‌وی بسته (%d) باید > NOTAM لغوشده (%d)", twy, cancelled)
	}
}

func TestQCodeExtract(t *testing.T) {
	// از خط Q در متن فرمت‌شده
	code := qcode.Extract("", "A1234/26 NOTAMN\nQ) OIIX/QMRLC/IV/NBO/A/000/999/", "")
	if code != "QMRLC" {
		t.Errorf("Extract=%q، انتظار QMRLC", code)
	}
	if qcode.Extract("", "no code here", "") != "" {
		t.Error("نباید کدی پیدا شود")
	}
}
