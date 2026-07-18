package model

import "time"

// نقش هر فرودگاه/بخش در پرواز (برای گروه‌بندی بریفینگ).
const (
	RoleADEP    = "ADEP"    // مبدأ
	RoleADES    = "ADES"    // مقصد
	RoleALTN    = "ALTN"    // الترنت
	RoleEnroute = "ENROUTE" // مسیر/FIR
)

// FlightPlan تعریف یک پرواز برای ساخت بریفینگ (E5-1).
// حدوداً ۵ فرودگاه: مبدأ + مقصد + الترنت‌ها؛ به‌علاوهٔ FIRهای مسیر و پنجرهٔ زمانی.
type FlightPlan struct {
	BaseModel
	Username string `gorm:"size:120;index;not null"`

	ADEP string `gorm:"size:4;index;not null"` // فرودگاه مبدأ
	ADES string `gorm:"size:4;index;not null"` // فرودگاه مقصد

	Alternates  StringSlice `gorm:"type:TEXT"` // کدهای ICAO الترنت
	EnrouteFIRs StringSlice `gorm:"type:TEXT"` // FIRهای مسیر برای تطبیق enroute

	ETD           time.Time `gorm:"type:TIMESTAMP with time zone;not null"` // زمان تخمینی حرکت
	ETA           time.Time `gorm:"type:TIMESTAMP with time zone;not null"` // زمان تخمینی رسیدن
	BufferMinutes int       `gorm:"default:60"`                             // حاشیهٔ اطمینان پنجرهٔ زمانی

	// بستر پرواز برای لایهٔ Operational Impact (E5.5)
	AircraftCategory string `gorm:"size:12;default:JET"` // JET / TURBOPROP / PISTON
	FlightRules      string `gorm:"size:4;default:IFR"`  // IFR / VFR

	// ارتفاع سِیر (fallback وقتی پروفایل نیست). ۰ = نامعلوم → UNKNOWN_FLIGHT_LEVEL.
	CruiseAltitudeFt int

	// مسیر واقعی و پروفایل ارتفاعی segment-based (E5.6+) — backward-compatible؛ nil = fallback.
	RouteWaypoints       Waypoints  `gorm:"type:TEXT"`
	RouteAltitudeProfile AltProfile `gorm:"type:TEXT"`

	Note string `gorm:"size:200"`
}

// دسته‌های هواپیما و قوانین پرواز.
const (
	AircraftJet       = "JET"
	AircraftTurboprop = "TURBOPROP"
	AircraftPiston    = "PISTON"
	RulesIFR          = "IFR"
	RulesVFR          = "VFR"
)

func (FlightPlan) TableName() string { return "flight_plans" }

// Airports مجموعهٔ فرودگاه‌های پرواز (مبدأ + مقصد + الترنت‌ها).
func (f FlightPlan) Airports() []string {
	out := make([]string, 0, 2+len(f.Alternates))
	if f.ADEP != "" {
		out = append(out, f.ADEP)
	}
	if f.ADES != "" {
		out = append(out, f.ADES)
	}
	out = append(out, f.Alternates...)
	return out
}

// Window پنجرهٔ زمانی مؤثر پرواز با احتساب buffer.
func (f FlightPlan) Window() (start, end time.Time) {
	b := time.Duration(f.BufferMinutes) * time.Minute
	return f.ETD.Add(-b), f.ETA.Add(b)
}
