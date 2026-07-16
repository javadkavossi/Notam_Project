// Package analysis هر NOTAM را دسته‌بندی و امتیاز اهمیت پایه می‌دهد (E3-2/E3-3/E3-4).
//
// امتیاز کاملاً قاعده‌محور و قابل‌توضیح است: امتیاز پایه از دستهٔ Q-code + تعدیل وضعیت +
// پرچم‌های بحرانی. این «امتیاز پایهٔ مستقل از پرواز» است؛ امتیاز کانتکستیِ وابسته به پرواز در E5.
//
// جدول وزن‌ها (نسخهٔ WeightsVersion) برای بازبینی کارشناس هوانوردی مستند شده:
// docs/phase1/QCODE_WEIGHTS.md (E3-5).
package analysis

import (
	"strings"

	"github.com/hossein-repo/BaseProject/internal/messaging"
	"github.com/hossein-repo/BaseProject/internal/pipeline/qcode"
)

// WeightsVersion نسخهٔ جدول وزن‌دهی (با هر تغییر در وزن‌ها افزایش یابد).
const WeightsVersion = "1.0.0"

// سطوح اهمیت.
const (
	LevelCritical = "CRITICAL"
	LevelHigh     = "HIGH"
	LevelMedium   = "MEDIUM"
	LevelLow      = "LOW"
	LevelInfo     = "INFO"
)

// فازهای پرواز.
const (
	PhaseDeparture = "DEPARTURE"
	PhaseEnroute   = "ENROUTE"
	PhaseApproach  = "APPROACH"
	PhaseLanding   = "LANDING"
	PhaseGround    = "GROUND"
)

// پرچم‌های ویژه.
const (
	TagFICON     = "FICON"
	TagRwyClosed = "RWY_CLOSED"
	TagAdClosed  = "AD_CLOSED"
	TagILSOut    = "ILS_OUT"
	TagGPSOut    = "GPS_OUT"
	TagObstacle  = "OBSTACLE"
)

// Result خروجی تحلیل یک NOTAM.
type Result struct {
	QCode          string
	Subject        string
	Condition      string
	Category       string
	SubjectLabel   string
	ConditionLabel string
	Phases         []string
	Tags           []string
	BaseScore      int
	BaseLevel      string
	FromText       bool // آیا از fallback متنی به‌دست آمد (Q-code نبود)
}

// وزن پایهٔ هر دسته (0..~80).
var categoryBase = map[string]int{
	qcode.CatAerodrome:   78,
	qcode.CatRunway:      70,
	qcode.CatILS:         50,
	qcode.CatGNSS:        50,
	qcode.CatNavigation:  45,
	qcode.CatLighting:    40,
	qcode.CatTaxiway:      38,
	qcode.CatMovementArea: 35, // موضوع دقیق ناشناخته در گروه M — عمداً پایین‌تر از باند
	qcode.CatApron:        25,
	qcode.CatAirspace:    55,
	qcode.CatRestriction: 55,
	qcode.CatProcedure:   50,
	qcode.CatComms:       40,
	qcode.CatWarning:     45,
	qcode.CatObstacle:    45,
	qcode.CatOther:       20,
}

// تعدیل امتیاز بر اساس وضعیت (۲ حرف).
var conditionDelta = map[string]int{
	// بستن/غیرقابل‌استفاده — بیشترین اهمیت
	"LC": 30, // Closed
	"LP": 27, // Prohibited
	"LD": 26, // Unsafe
	"AS": 22, // Unserviceable
	"AU": 22, // Not available
	"AW": 20, // Withdrawn
	"LI": 20, // Closed to IFR
	"LV": 10, // Closed to VFR
	"LN": 12, // Closed to night ops
	// محدودیت‌های جزئی
	"LT": 8,  // Limited to
	"LL": 8,  // Usable limited length/width
	"CP": 10, // Reduced power
	"CG": 12, // Downgraded
	"CM": 10, // Displaced
	"CL": 10, // Realigned
	"LS": 8,  // Subject to interruption
	// تغییرات فعال/خطر
	"CA": 12, // Activated
	"CE": 12, // Erected
	"CS": 8,  // Installed
	"CT": 15, // On test, do not use
	"HW": 10, // Work in progress
	"HH": 20, // Hazard
	// اخبار خوب/کاهش اهمیت
	"CN": -50, // Cancelled
	"AO": -18, // Operational
	"AK": -18, // Resumed
	"CC": -12, // Completed
	"HV": -12, // Work completed
}

// نگاشت دسته → فازهای پرواز مرتبط.
var categoryPhases = map[string][]string{
	qcode.CatAerodrome:   {PhaseDeparture, PhaseLanding, PhaseGround},
	qcode.CatRunway:       {PhaseDeparture, PhaseLanding},
	qcode.CatTaxiway:      {PhaseGround},
	qcode.CatMovementArea: {PhaseGround},
	qcode.CatApron:       {PhaseGround},
	qcode.CatLighting:    {PhaseApproach, PhaseLanding},
	qcode.CatILS:         {PhaseApproach, PhaseLanding},
	qcode.CatNavigation:  {PhaseEnroute, PhaseApproach},
	qcode.CatGNSS:        {PhaseEnroute, PhaseApproach},
	qcode.CatComms:       {PhaseEnroute, PhaseApproach},
	qcode.CatAirspace:    {PhaseEnroute},
	qcode.CatRestriction: {PhaseEnroute},
	qcode.CatProcedure:   {PhaseDeparture, PhaseApproach},
	qcode.CatWarning:     {PhaseEnroute},
	qcode.CatObstacle:    {PhaseApproach, PhaseDeparture},
	qcode.CatOther:       {},
}

// Analyze یک NOTAM را تحلیل می‌کند.
func Analyze(ev messaging.NotamEvent) Result {
	code := qcode.Extract(ev.QCode, ev.HumanReadableText, ev.Text)
	if code == "" {
		return analyzeFromText(ev)
	}
	d := qcode.Decode(code)

	res := Result{
		QCode:          d.Code,
		Subject:        d.Subject,
		Condition:      d.Condition,
		Category:       d.Category,
		SubjectLabel:   d.SubjectLabel,
		ConditionLabel: d.ConditionLabel,
		Phases:         categoryPhases[d.Category],
	}

	score := categoryBase[d.Category]
	if delta, ok := conditionDelta[d.Condition]; ok {
		score += delta
	}

	// برچسب‌های وابسته به دسته فقط وقتی صادر می‌شوند که موضوع Q-code دقیقاً شناسایی شده باشد.
	// اگر موضوع حدس زده شده (fallback حرف اول)، برچسبِ قاطع مثل RWY_CLOSED گمراه‌کننده است.
	res.Tags = deriveTags(d.Category, d.Condition, ev.Text+" "+ev.HumanReadableText, d.Recognized)
	score += tagBonus(res.Tags)

	res.BaseScore = clamp(score)
	res.BaseLevel = level(res.BaseScore)
	return res
}

// analyzeFromText وقتی Q-code نیست: تحلیل بر اساس کلیدواژه‌های متن (E3-2).
func analyzeFromText(ev messaging.NotamEvent) Result {
	text := strings.ToUpper(ev.Text + " " + ev.HumanReadableText)
	res := Result{FromText: true, Category: qcode.CatOther}

	switch {
	case strings.Contains(text, "FICON"):
		res.Category = qcode.CatRunway
		res.BaseScore = 65
	case containsAll(text, "RWY", "CLSD") || containsAll(text, "RWY", "CLOSED") || strings.Contains(text, "RUNWAY CLOSED"):
		res.Category = qcode.CatRunway
		res.BaseScore = 85
	case strings.Contains(text, "AD CLSD") || strings.Contains(text, "AERODROME CLOSED") || strings.Contains(text, "AIRPORT CLOSED"):
		res.Category = qcode.CatAerodrome
		res.BaseScore = 90
	case strings.Contains(text, "ILS") && unserviceable(text):
		res.Category = qcode.CatILS
		res.BaseScore = 70
	case (strings.Contains(text, "GPS") || strings.Contains(text, "RAIM") || strings.Contains(text, "GNSS")) && (unserviceable(text) || strings.Contains(text, "OUTAGE")):
		res.Category = qcode.CatGNSS
		res.BaseScore = 62
	case strings.Contains(text, "OBST") || strings.Contains(text, "CRANE"):
		res.Category = qcode.CatObstacle
		res.BaseScore = 45
	case containsAll(text, "TWY", "CLSD") || containsAll(text, "TWY", "CLOSED"):
		res.Category = qcode.CatTaxiway
		res.BaseScore = 45
	default:
		res.BaseScore = 20
	}

	res.Phases = categoryPhases[res.Category]
	// در این مسیر دسته مستقیماً از کلیدواژه‌های متن آمده (نه حدسِ Q-code)، پس برچسب‌ها معتبرند.
	res.Tags = deriveTags(res.Category, "", text, true)
	res.BaseScore = clamp(res.BaseScore + tagBonus(res.Tags))
	res.BaseLevel = level(res.BaseScore)
	return res
}

// deriveTags برچسب‌های بحرانی را استخراج می‌کند.
// subjectKnown: آیا موضوع Q-code دقیقاً شناسایی شده؟ اگر نه، برچسب‌های وابسته به دسته
// صادر نمی‌شوند (فقط برچسب‌های متن‌محور مثل FICON که مستقل از Q-code قابل اتکا هستند).
func deriveTags(category, condition, text string, subjectKnown bool) []string {
	up := strings.ToUpper(text)
	var tags []string
	closed := condition == "LC" || strings.Contains(up, "CLSD") || strings.Contains(up, "CLOSED")

	// متن‌محور: مستقل از شناسایی Q-code معتبر است
	if strings.Contains(up, "FICON") {
		tags = append(tags, TagFICON)
	}
	if !subjectKnown {
		return tags
	}

	if category == qcode.CatRunway && closed {
		tags = append(tags, TagRwyClosed)
	}
	if category == qcode.CatAerodrome && closed {
		tags = append(tags, TagAdClosed)
	}
	if category == qcode.CatILS && (condition == "AS" || condition == "AU" || unserviceable(up)) {
		tags = append(tags, TagILSOut)
	}
	if category == qcode.CatGNSS && (condition == "AS" || condition == "AU" || unserviceable(up) || strings.Contains(up, "OUTAGE")) {
		tags = append(tags, TagGPSOut)
	}
	if category == qcode.CatObstacle {
		tags = append(tags, TagObstacle)
	}
	return tags
}

// tagBonus امتیاز اضافی برای پرچم‌های بحرانی.
func tagBonus(tags []string) int {
	bonus := 0
	for _, t := range tags {
		switch t {
		case TagAdClosed:
			bonus += 12
		case TagRwyClosed:
			bonus += 10
		case TagFICON:
			bonus += 8
		case TagILSOut, TagGPSOut:
			bonus += 6
		case TagObstacle:
			bonus += 3
		}
	}
	return bonus
}

// LevelFor سطح اهمیت را از امتیاز محاسبه می‌کند (همان آستانه‌های امتیاز پایه).
// موتور بریفینگ برای امتیاز کانتکستی از همین تابع استفاده می‌کند تا سطوح یکسان بمانند.
func LevelFor(score int) string { return level(score) }

// Clamp امتیاز را در بازهٔ ۰..۱۰۰ نگه می‌دارد.
func Clamp(score int) int { return clamp(score) }

func level(score int) string {
	switch {
	case score >= 80:
		return LevelCritical
	case score >= 60:
		return LevelHigh
	case score >= 35:
		return LevelMedium
	case score >= 15:
		return LevelLow
	default:
		return LevelInfo
	}
}

func clamp(s int) int {
	if s < 0 {
		return 0
	}
	if s > 100 {
		return 100
	}
	return s
}

func unserviceable(up string) bool {
	return strings.Contains(up, "U/S") || strings.Contains(up, "UNSERVICEABLE") || strings.Contains(up, "UNSVC") || strings.Contains(up, "OUT OF SERVICE")
}

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		if !strings.Contains(s, sub) {
			return false
		}
	}
	return true
}
