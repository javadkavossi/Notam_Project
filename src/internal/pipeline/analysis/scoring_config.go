package analysis

import "github.com/hossein-repo/BaseProject/internal/pipeline/qcode"

// scoring_config.go — تنظیمات نسخه‌بندی‌شدهٔ امتیازدهی (E3-calibration).
//
// اصل: وزن دسته = «سقف بالقوه / بیشترین شدت عملیاتی» (Base Potential)، نه امتیاز نهایی.
// امتیاز نهایی از جریان زیر می‌آید: Evidence → Base Potential → Operational Impact → Flight Context.
//
// هر نتیجه ScoringVersion دارد تا با نسخهٔ همان config قابل بازتولید باشد. نسخه‌های قبلی
// حذف نمی‌شوند (baseline حفظ می‌شود) تا مقایسه و reproducibility ممکن بماند.

// Thresholds آستانهٔ سطوح اهمیت.
type Thresholds struct {
	Critical int // >= → CRITICAL
	High     int // >= → HIGH
	Medium   int // >= → MEDIUM
	Low      int // >= → LOW ، وگرنه INFO
}

// ScoringConfig مجموعهٔ کاملِ وزن‌ها و آستانه‌ها برای یک نسخه.
type ScoringConfig struct {
	Version       string
	EffectiveFrom string

	// CategoryCeilings سقفِ بالقوهٔ هر دسته (بدترین شرایط ممکن؛ Context آن را کاهش می‌دهد).
	CategoryCeilings map[string]int
	ConditionDeltas  map[string]int
	TagBonuses       map[string]int
	Thresholds       Thresholds

	// RouteCorridorNM نصف‌عرض کریدور مسیر (NM) — از hard-code خارج شد (بدون تغییر الگوریتم).
	RouteCorridorNM float64

	// سقف‌های عدم‌تشدید در لایهٔ Impact (تا داده نامعلوم/بی‌ربط هرگز خودکار HIGH/CRITICAL نشود).
	NotApplicableCap int // سقف امتیازِ NOT_APPLICABLE
	UnknownCap       int // سقف امتیازِ حالت‌های نامعلوم (زیر HIGH)
}

// CategoryBase سقف بالقوهٔ یک دسته را برمی‌گرداند.
func (c *ScoringConfig) CategoryBase(cat string) int { return c.CategoryCeilings[cat] }

// ConditionDelta تعدیل وضعیت را برمی‌گرداند (۰ اگر نبود).
func (c *ScoringConfig) ConditionDelta(cond string) int { return c.ConditionDeltas[cond] }

// Level سطح را از امتیاز طبق آستانه‌های همین نسخه محاسبه می‌کند.
func (c *ScoringConfig) Level(score int) string {
	switch {
	case score >= c.Thresholds.Critical:
		return LevelCritical
	case score >= c.Thresholds.High:
		return LevelHigh
	case score >= c.Thresholds.Medium:
		return LevelMedium
	case score >= c.Thresholds.Low:
		return LevelLow
	default:
		return LevelInfo
	}
}

// ---------------------------------------------------------------------------
// نسخهٔ 1.1.0 — baseline (وزن‌های مهندسی اولیه + کالیبراسیون ایمنِ کارشناس: CN=-40، TT=-45)
// ---------------------------------------------------------------------------

var configV110 = &ScoringConfig{
	Version:       "1.1.0",
	EffectiveFrom: "2026-07-17",
	CategoryCeilings: map[string]int{
		qcode.CatAerodrome: 78, qcode.CatRunway: 70, qcode.CatILS: 50, qcode.CatGNSS: 50,
		qcode.CatNavigation: 45, qcode.CatLighting: 40, qcode.CatTaxiway: 38, qcode.CatMovementArea: 35,
		qcode.CatApron: 25, qcode.CatAirspace: 55, qcode.CatRestriction: 55, qcode.CatATS: 55,
		qcode.CatRescue: 62, qcode.CatProcedure: 50, qcode.CatMet: 45, qcode.CatComms: 40,
		qcode.CatWarning: 45, qcode.CatObstacle: 45, qcode.CatService: 28, qcode.CatOther: 20,
	},
	ConditionDeltas:  baselineConditionDeltas(),
	TagBonuses:       baselineTagBonuses(),
	Thresholds:       Thresholds{Critical: 80, High: 60, Medium: 35, Low: 15},
	RouteCorridorNM:  25,
	NotApplicableCap: 20,
	UnknownCap:       59,
}

// ---------------------------------------------------------------------------
// نسخهٔ 1.2.0 — سقف‌های بالقوهٔ کارشناس دیسپچ (بازبینی روی EXPERT_REVIEW).
// این اعداد «سقف بالقوه»اند؛ لایهٔ Operational Impact آن‌ها را به امتیاز نهایی می‌رساند.
// جزئیات و موارد اختلافی: docs/phase1/CALIBRATION_v1.2.0.md
// ---------------------------------------------------------------------------

var configV120 = &ScoringConfig{
	Version:       "1.2.0",
	EffectiveFrom: "2026-07-19",
	CategoryCeilings: map[string]int{
		qcode.CatAerodrome:    70, // 78→70 (کارشناس)
		qcode.CatRunway:       70, // بدون تغییر
		qcode.CatRescue:       62, // بدون تغییر (نبود RFFS ≈ شرط پیش‌پرواز)
		qcode.CatAirspace:     60, // 55→60
		qcode.CatRestriction:  60, // 55→60
		qcode.CatATS:          67, // 55→67 (برج/نزدیکی/رادار)
		qcode.CatILS:          55, // 50→55
		qcode.CatProcedure:    55, // 50→55
		qcode.CatGNSS:         50, // بدون تغییر
		qcode.CatMet:          70, // 45→70 (RVR/باد در عملیات کم‌دید بحرانی)
		qcode.CatWarning:      45, // بدون تغییر
		qcode.CatObstacle:     40, // 45→40
		qcode.CatNavigation:   60, // 45→60
		qcode.CatLighting:     47, // 40→47
		qcode.CatComms:        60, // 40→60
		qcode.CatTaxiway:      35, // 38→35
		qcode.CatMovementArea: 35, // بدون تغییر
		qcode.CatService:      37, // 28→37 (سوخت/اکسیژن/گمرک)
		qcode.CatApron:        25, // بازبینی‌نشده توسط کارشناس → بدون تغییر
		qcode.CatOther:        25, // 20→25
	},
	ConditionDeltas:  baselineConditionDeltas(),                               // تأییدشده توسط کارشناس (به‌جز CN/TT که قبلاً اعمال شد)
	TagBonuses:       baselineTagBonuses(),                                    // فهرست کامل ۲۴ پرچم کارشناس در فاز جدا
	Thresholds:       Thresholds{Critical: 80, High: 60, Medium: 35, Low: 15}, // بدون تحمیل مصنوعیِ درصد
	RouteCorridorNM:  25,
	NotApplicableCap: 20,
	UnknownCap:       59,
}

func baselineConditionDeltas() map[string]int {
	return map[string]int{
		"LC": 30, "LP": 27, "LD": 26, "AS": 22, "AU": 22, "AW": 20, "LI": 20, "HH": 20,
		"CT": 15, "CG": 12, "CA": 12, "CE": 12, "LN": 12,
		"CM": 10, "CL": 10, "LV": 10, "HW": 10, "CP": 10,
		"LT": 8, "LL": 8, "LS": 8, "CS": 8,
		"CN": -40, // Cancelled (کارشناس: −۴۰)
		"AO": -18, "AK": -18, "CC": -12, "HV": -12,
		"TT": -45, // Trigger NOTAM (کارشناس: اطلاعی)
	}
}

func baselineTagBonuses() map[string]int {
	return map[string]int{
		TagAdClosed: 12, TagRwyClosed: 10, TagFICON: 8, TagILSOut: 6, TagGPSOut: 6, TagObstacle: 3,
	}
}

// registry همهٔ نسخه‌ها برای بازتولید (reproducibility).
var registry = map[string]*ScoringConfig{
	configV110.Version: configV110,
	configV120.Version: configV120,
}

// Current نسخهٔ فعالِ امتیازدهی.
var Current = configV120

// ConfigByVersion یک نسخهٔ مشخص را برمی‌گرداند (nil اگر نبود) — برای بازتولید نتایج قدیمی.
func ConfigByVersion(v string) *ScoringConfig { return registry[v] }

// CurrentConfig نسخهٔ فعال را برمی‌گرداند (برای پکیج‌های دیگر مثل briefing).
func CurrentConfig() *ScoringConfig { return Current }
