// Package qcode دیکد کد Q استاندارد ICAO را انجام می‌دهد (E3-1).
//
// ساختار Q-code: Q + دو حرف «موضوع» (subject) + دو حرف «وضعیت» (condition). مثال: QMRLC
//   MR = Runway (موضوع) ، LC = Closed (وضعیت)  →  «باند بسته»
//
// دیکد کاملاً deterministic و قابل‌ممیزی است (بدون LLM) — پایهٔ دقتِ سیستم.
// مرجع: ICAO Doc 8126 / Annex 15.
package qcode

import (
	"regexp"
	"strings"
)

// دسته‌های اصلی (Category) که موضوع Q-code به آن نگاشت می‌شود.
const (
	CatAerodrome = "AERODROME"
	CatRunway    = "RUNWAY"
	CatTaxiway   = "TAXIWAY"
	CatApron     = "APRON"
	// CatMovementArea برای کدهای گروه M که موضوع دقیقشان شناخته نشده.
	// عمداً از RUNWAY جدا است: نسبت‌دادن «باند» به یک کد ناشناخته باعث تشدید
	// خطرناکِ اهمیت می‌شود (مثلاً بستن تاکسی‌وی → «باند بسته»).
	CatMovementArea = "MOVEMENT_AREA"
	CatLighting    = "LIGHTING"
	CatILS         = "ILS"
	CatNavigation  = "NAVIGATION"
	CatGNSS        = "GNSS"
	CatComms       = "COMMS"
	CatAirspace    = "AIRSPACE"
	CatRestriction = "RESTRICTION"
	CatProcedure   = "PROCEDURE"
	CatWarning     = "WARNING"
	CatObstacle    = "OBSTACLE"
	CatOther       = "OTHER"
)

// Decoded نتیجهٔ دیکد یک Q-code.
type Decoded struct {
	Code           string // کد کامل مثل "QMRLC"
	Subject        string // دو حرف موضوع "MR"
	Condition      string // دو حرف وضعیت "LC"
	Category       string // یکی از Cat*
	SubjectLabel   string // توضیح انسانی موضوع
	ConditionLabel string // توضیح انسانی وضعیت
	Recognized     bool   // آیا کد معتبر شناسایی شد
}

type subjectInfo struct {
	category string
	label    string
}

// subjectMap موضوع (۲ حرف) → دسته + برچسب. زیرمجموعه‌ای نمایندهٔ جدول ICAO.
var subjectMap = map[string]subjectInfo{
	// Movement area (M) — سطح حرکت و فرود.
	// ⚠️ دقت در تفکیک باند/تاکسی‌وی حیاتی است: نسبت‌دادن اشتباهِ «باند» به تاکسی‌وی
	// اهمیت را به‌غلط تا سطح بحرانی بالا می‌برد.
	"MR": {CatRunway, "Runway"},
	"MS": {CatRunway, "Stopway"},
	"MT": {CatRunway, "Threshold"},
	"MU": {CatRunway, "Runway turning bay"},
	"MW": {CatRunway, "Strip/shoulder"},
	"MD": {CatRunway, "Declared distances"},
	"MH": {CatRunway, "Runway arresting gear"},
	"MX": {CatTaxiway, "Taxiway(s)"},
	"MY": {CatTaxiway, "Rapid exit taxiway"},
	"MG": {CatTaxiway, "Taxiing guidance system"},
	"MK": {CatApron, "Parking area"},
	"MN": {CatApron, "Apron"},
	"MP": {CatApron, "Aircraft stands"},
	"MO": {CatMovementArea, "Stopbar"},
	"MA": {CatMovementArea, "Movement area"},
	"MB": {CatMovementArea, "Bearing strength"},
	"MC": {CatMovementArea, "Clearway"},
	"MM": {CatMovementArea, "Daylight markings"},
	// Facilities (F)
	"FA": {CatAerodrome, "Aerodrome"},
	"FF": {CatAerodrome, "Fire and rescue"},
	"FU": {CatAerodrome, "Fuel availability"},
	// Lighting (L)
	"LA": {CatLighting, "Approach lighting system"},
	"LB": {CatLighting, "Aerodrome beacon"},
	"LC": {CatLighting, "Runway centre line lights"},
	"LE": {CatLighting, "Runway edge lights"},
	"LH": {CatLighting, "High intensity runway lights"},
	"LI": {CatLighting, "Runway end identifier lights"},
	"LP": {CatLighting, "PAPI"},
	"LR": {CatLighting, "All landing area lighting"},
	"LT": {CatLighting, "Threshold lights"},
	"LV": {CatLighting, "VASIS"},
	// ILS / MLS (I)
	"IC": {CatILS, "ILS"},
	"ID": {CatILS, "DME associated with ILS"},
	"IG": {CatILS, "Glide path (ILS)"},
	"II": {CatILS, "Inner marker (ILS)"},
	"IL": {CatILS, "Localizer (ILS)"},
	"IM": {CatILS, "Middle marker (ILS)"},
	"IO": {CatILS, "Outer marker (ILS)"},
	"IS": {CatILS, "ILS Category I"},
	"IT": {CatILS, "ILS Category II"},
	"IU": {CatILS, "ILS Category III"},
	"IW": {CatILS, "MLS"},
	// GNSS (G)
	"GA": {CatGNSS, "GNSS airfield-specific operations"},
	"GW": {CatGNSS, "GNSS area-wide operations"},
	// Terminal/en-route navigation (N)
	"NA": {CatNavigation, "All radio navigation facilities"},
	"NB": {CatNavigation, "NDB"},
	"ND": {CatNavigation, "DME"},
	"NL": {CatNavigation, "Locator"},
	"NM": {CatNavigation, "VOR/DME"},
	"NN": {CatNavigation, "NDB"},
	"NT": {CatNavigation, "VORTAC"},
	"NV": {CatNavigation, "VOR"},
	"NX": {CatNavigation, "TACAN"},
	// Communications / surveillance (C)
	"CA": {CatComms, "Air/ground facility"},
	"CE": {CatComms, "En route surveillance radar"},
	"CM": {CatComms, "Surface movement radar"},
	"CP": {CatComms, "Precision approach radar"},
	"CS": {CatComms, "Secondary surveillance radar"},
	"CT": {CatComms, "Terminal area surveillance radar"},
	// Airspace organization (A)
	"AC": {CatAirspace, "Class B/C/D/E surface area"},
	"AD": {CatAirspace, "Air defense identification zone"},
	"AE": {CatAirspace, "Control area"},
	"AF": {CatAirspace, "Flight information region"},
	"AH": {CatAirspace, "Upper control area"},
	"AN": {CatAirspace, "Area navigation route"},
	"AR": {CatAirspace, "ATS route"},
	"AT": {CatAirspace, "Terminal control area"},
	"AZ": {CatAirspace, "Aerodrome traffic zone"},
	// Restrictions (R)
	"RA": {CatRestriction, "Airspace reservation"},
	"RD": {CatRestriction, "Danger area"},
	"RM": {CatRestriction, "Military operating area"},
	"RP": {CatRestriction, "Prohibited area"},
	"RR": {CatRestriction, "Restricted area"},
	"RT": {CatRestriction, "Temporary restricted area"},
	// ATM procedures (P)
	"PA": {CatProcedure, "Standard instrument arrival (STAR)"},
	"PD": {CatProcedure, "Standard instrument departure (SID)"},
	"PI": {CatProcedure, "Instrument approach procedure"},
	"PM": {CatProcedure, "Aerodrome operating minima"},
	"PU": {CatProcedure, "Missed approach procedure"},
	"PT": {CatProcedure, "Transition altitude"},
	// Warnings (W)
	"WM": {CatWarning, "Missile/gun/rocket firing"},
	"WP": {CatWarning, "Parachute jumping"},
	"WU": {CatWarning, "Unmanned aircraft"},
	"WW": {CatWarning, "Significant volcanic activity"},
	// Other (O) — شامل موانع
	"OB": {CatObstacle, "Obstacle"},
	"OL": {CatObstacle, "Obstacle lights"},
	"OA": {CatOther, "Aeronautical information service"},
	"OE": {CatOther, "Aircraft entry requirements"},
}

// subjectByFirstLetter نگاشت خشن بر اساس حرف اول موضوع، برای کدهای ناشناخته (fallback).
// اصل: fallback هرگز نباید به پرخطرترین دستهٔ گروه نگاشت شود؛ در تردید، دستهٔ خنثی‌تر
// انتخاب می‌شود تا اهمیت به‌غلط تشدید نشود (Recognized=false هم ثبت می‌شود).
var subjectByFirstLetter = map[byte]subjectInfo{
	'M': {CatMovementArea, "Movement area"},
	'F': {CatAerodrome, "Facility/service"},
	'L': {CatLighting, "Lighting facility"},
	'I': {CatILS, "ILS/MLS"},
	'G': {CatGNSS, "GNSS service"},
	'N': {CatNavigation, "Navigation facility"},
	'C': {CatComms, "Communications/surveillance"},
	'A': {CatAirspace, "Airspace organization"},
	'R': {CatRestriction, "Airspace restriction"},
	'P': {CatProcedure, "ATM procedure"},
	'W': {CatWarning, "Navigation warning"},
	'O': {CatOther, "Other information"},
}

// conditionMap وضعیت (۲ حرف) → برچسب. زیرمجموعهٔ رایج جدول ICAO.
var conditionMap = map[string]string{
	// Availability (A)
	"AS": "Unserviceable",
	"AU": "Not available",
	"AW": "Withdrawn",
	"AO": "Operational",
	"AK": "Resumed normal operation",
	"AL": "Operative subject to limitations",
	"AP": "Available, prior permission required",
	"AR": "Available on request",
	"AC": "Withdrawn for maintenance",
	// Changes (C)
	"CA": "Activated",
	"CC": "Completed",
	"CD": "Deactivated",
	"CE": "Erected",
	"CF": "Frequency changed",
	"CG": "Downgraded",
	"CH": "Changed",
	"CL": "Realigned",
	"CM": "Displaced",
	"CN": "Cancelled",
	"CO": "Operating",
	"CP": "Operating on reduced power",
	"CR": "Temporarily replaced",
	"CS": "Installed",
	"CT": "On test, do not use",
	// Hazard (H)
	"HW": "Work in progress",
	"HV": "Work completed",
	"HX": "Concentration of snow",
	"HH": "Hazard",
	// Limitations (L)
	"LC": "Closed",
	"LD": "Unsafe",
	"LI": "Closed to IFR operations",
	"LL": "Usable, limited length/width",
	"LN": "Closed to night operations",
	"LP": "Prohibited",
	"LR": "Aircraft restricted to runways/taxiways",
	"LS": "Subject to interruption",
	"LT": "Limited to",
	"LV": "Closed to VFR operations",
	"LW": "Will take place",
	// Trigger / Other
	"TT": "Trigger NOTAM",
	"XX": "Plain language",
}

var qcodeRe = regexp.MustCompile(`\bQ([A-Z]{4})\b`)

// Extract نخستین Q-code معتبر را از متن‌های داده‌شده استخراج می‌کند (به‌ترتیب اولویت).
// معمولاً از فیلد qcode، سپس خط Q در متن فرمت‌شده، سپس متن اصلی.
func Extract(texts ...string) string {
	for _, t := range texts {
		t = strings.ToUpper(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		// اگر خودِ کد پنج‌حرفی داده شده
		if len(t) == 5 && t[0] == 'Q' {
			return t
		}
		if m := qcodeRe.FindStringSubmatch(t); len(m) >= 2 {
			return "Q" + m[1]
		}
	}
	return ""
}

// Decode یک Q-code را به ساختار معنایی تبدیل می‌کند.
func Decode(code string) Decoded {
	code = strings.ToUpper(strings.TrimSpace(code))
	d := Decoded{Code: code}
	if len(code) != 5 || code[0] != 'Q' {
		d.Category = CatOther
		return d
	}
	d.Subject = code[1:3]
	d.Condition = code[3:5]

	if si, ok := subjectMap[d.Subject]; ok {
		d.Category = si.category
		d.SubjectLabel = si.label
		d.Recognized = true
	} else if si, ok := subjectByFirstLetter[d.Subject[0]]; ok {
		d.Category = si.category
		d.SubjectLabel = si.label // برچسب خشن
	} else {
		d.Category = CatOther
	}

	if cl, ok := conditionMap[d.Condition]; ok {
		d.ConditionLabel = cl
	} else {
		d.ConditionLabel = conditionByFirstLetter(d.Condition[0])
	}
	return d
}

func conditionByFirstLetter(c byte) string {
	switch c {
	case 'A':
		return "Availability change"
	case 'C':
		return "Change"
	case 'H':
		return "Hazard condition"
	case 'L':
		return "Limitation"
	case 'T':
		return "Trigger"
	default:
		return ""
	}
}
