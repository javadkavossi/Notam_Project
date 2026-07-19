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

// WeightsVersion نسخهٔ فعالِ امتیازدهی (از config؛ برای بازتولید در خروجی ذخیره می‌شود).
var WeightsVersion = Current.Version

// وضعیت corpus (بدون FlightPlan): آیا این NOTAM به‌تنهایی قابل‌قضاوت است یا نیاز به بستر پرواز دارد.
const (
	CorpusBaseOnly          = "BASE_ONLY"          // امتیاز پایه تقریباً نهایی است (مثل بستن کامل فرودگاه)
	CorpusContextRequired   = "CONTEXT_REQUIRED"   // امتیاز نهایی نیازمند بستر پرواز است
	CorpusInformationalOnly = "INFORMATIONAL_ONLY" // Trigger/Cancelled/اطلاعی
)

// سطوح اطمینانِ تحلیل پایه.
const (
	ConfidenceHigh   = "HIGH"
	ConfidenceMedium = "MEDIUM"
	ConfidenceLow    = "LOW"
)

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
	BaseScore      int    // Base Potential (سقف بالقوه؛ نه امتیاز نهایی)
	BaseLevel      string // بندِ بالقوه (نه priority نهایی)
	FromText       bool   // آیا از fallback متنی به‌دست آمد (Q-code نبود)

	// کالیبراسیون (E3-calibration)
	ScoringVersion string // نسخهٔ config که این نتیجه با آن ساخته شد (برای بازتولید)
	CorpusStatus   string // BASE_ONLY / CONTEXT_REQUIRED / INFORMATIONAL_ONLY
	Confidence     string // HIGH (Q-code شناسایی‌شده) / MEDIUM (متن) / LOW (حدس)
	MissingData    []string
}

// نگاشت دسته → فازهای پرواز مرتبط.
var categoryPhases = map[string][]string{
	qcode.CatAerodrome:    {PhaseDeparture, PhaseLanding, PhaseGround},
	qcode.CatRunway:       {PhaseDeparture, PhaseLanding},
	qcode.CatTaxiway:      {PhaseGround},
	qcode.CatMovementArea: {PhaseGround},
	qcode.CatApron:        {PhaseGround},
	qcode.CatLighting:     {PhaseApproach, PhaseLanding},
	qcode.CatILS:          {PhaseApproach, PhaseLanding},
	qcode.CatNavigation:   {PhaseEnroute, PhaseApproach},
	qcode.CatGNSS:         {PhaseEnroute, PhaseApproach},
	qcode.CatComms:        {PhaseEnroute, PhaseApproach},
	qcode.CatATS:          {PhaseDeparture, PhaseEnroute, PhaseApproach},
	qcode.CatRescue:       {PhaseDeparture, PhaseLanding},
	qcode.CatMet:          {PhaseApproach, PhaseLanding},
	qcode.CatService:      {PhaseGround},
	qcode.CatAirspace:     {PhaseEnroute},
	qcode.CatRestriction:  {PhaseEnroute},
	qcode.CatProcedure:    {PhaseDeparture, PhaseApproach},
	qcode.CatWarning:      {PhaseEnroute},
	qcode.CatObstacle:     {PhaseApproach, PhaseDeparture},
	qcode.CatOther:        {},
}

// Analyze یک NOTAM را با نسخهٔ فعالِ config تحلیل می‌کند.
func Analyze(ev messaging.NotamEvent) Result { return AnalyzeWith(ev, Current) }

// AnalyzeWith یک NOTAM را با یک نسخهٔ مشخصِ config تحلیل می‌کند (برای بازتولید نتایج قدیمی).
func AnalyzeWith(ev messaging.NotamEvent, cfg *ScoringConfig) Result {
	if cfg == nil {
		cfg = Current
	}
	code := qcode.Extract(ev.QCode, ev.HumanReadableText, ev.Text)
	if code == "" {
		return finalizeCorpus(analyzeFromText(ev, cfg), cfg)
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

	score := cfg.CategoryBase(d.Category)
	score += cfg.ConditionDelta(d.Condition)

	// برچسب‌های وابسته به دسته فقط وقتی صادر می‌شوند که موضوع Q-code دقیقاً شناسایی شده باشد.
	// اگر موضوع حدس زده شده (fallback حرف اول)، برچسبِ قاطع مثل RWY_CLOSED گمراه‌کننده است.
	res.Tags = deriveTags(d.Category, d.Condition, ev.Text+" "+ev.HumanReadableText, d.Recognized)
	score += tagBonus(res.Tags, cfg)

	res.BaseScore = clamp(score)
	res.BaseLevel = cfg.Level(res.BaseScore)
	// اطمینان: Q-code شناسایی‌شده → HIGH ، حدسِ حرف اول → LOW
	if d.Recognized {
		res.Confidence = ConfidenceHigh
	} else {
		res.Confidence = ConfidenceLow
	}
	return finalizeCorpus(res, cfg)
}

// finalizeCorpus وضعیت corpus و نسخه را تعیین می‌کند (بدون FlightPlan، امتیاز نهایی جعل نمی‌شود).
func finalizeCorpus(res Result, cfg *ScoringConfig) Result {
	res.ScoringVersion = cfg.Version
	res.CorpusStatus = corpusStatus(res)
	if res.CorpusStatus == CorpusContextRequired {
		res.MissingData = append(res.MissingData, "flightPlan")
	}
	if res.Confidence == "" {
		res.Confidence = ConfidenceMedium
	}
	return res
}

// corpusStatus بدون FlightPlan تعیین می‌کند این NOTAM چه وضعیتی دارد.
func corpusStatus(res Result) string {
	if res.Condition == "TT" || res.Condition == "CN" {
		return CorpusInformationalOnly
	}
	// بستن کاملِ فرودگاه تقریباً مستقل از بستر است
	if hasTagStr(res.Tags, TagAdClosed) {
		return CorpusBaseOnly
	}
	return CorpusContextRequired
}

func hasTagStr(tags []string, t string) bool {
	for _, x := range tags {
		if x == t {
			return true
		}
	}
	return false
}

// analyzeFromText وقتی Q-code نیست: تحلیل بر اساس کلیدواژه‌های متن (E3-2).
func analyzeFromText(ev messaging.NotamEvent, cfg *ScoringConfig) Result {
	text := strings.ToUpper(ev.Text + " " + ev.HumanReadableText)
	res := Result{FromText: true, Category: qcode.CatOther, Confidence: ConfidenceMedium}

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
	res.BaseScore = clamp(res.BaseScore + tagBonus(res.Tags, cfg))
	res.BaseLevel = cfg.Level(res.BaseScore)
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

// tagBonus امتیاز اضافی پرچم‌های بحرانی را از config می‌خواند.
func tagBonus(tags []string, cfg *ScoringConfig) int {
	bonus := 0
	for _, t := range tags {
		bonus += cfg.TagBonuses[t]
	}
	return bonus
}

// LevelFor سطح اهمیت را از امتیاز طبق آستانه‌های نسخهٔ فعال محاسبه می‌کند.
// موتور بریفینگ برای امتیاز کانتکستی از همین تابع استفاده می‌کند تا سطوح یکسان بمانند.
func LevelFor(score int) string { return Current.Level(score) }

// Clamp امتیاز را در بازهٔ ۰..۱۰۰ نگه می‌دارد.
func Clamp(score int) int { return clamp(score) }

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
