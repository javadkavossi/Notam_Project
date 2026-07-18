package model

import "time"

// Notam ذخیره NOTAM مطابق استاندارد ICAO و سازگار با خروجی Jeppesen
// داده از FAA SWIM/AIM FNS دریافت و به صورت ساختاری ذخیره می‌شود
type Notam struct {
	BaseModel

	// شناسه یکتا از منبع (Solace Message ID) — برای provenance و ردیابی
	MessageID string `gorm:"size:120;index;not null"`

	// CanonicalKey هویت متعارف NOTAM بین منابع مختلف (location|series). کلید یکتای واقعی (E2-3).
	CanonicalKey string `gorm:"size:140;uniqueIndex"`

	// Source منبع دریافت این NOTAM (FAA_SWIM/AFTN/…) — گام اول provenance (E1-4)
	Source string `gorm:"size:20;index"`

	// ICAO Standard Fields (برای خروجی استاندارد خلبان)
	SeriesNumber string `gorm:"size:20;index"`   // 0046/26, A3910/25
	EventType    string `gorm:"size:4;index"`    // N, R, C (new, replacement, cancel)
	QLine        string `gorm:"size:80"`         // Q) FIR/FIR/type/traffic/scope/...
	LocationICAO string `gorm:"size:8;index"`    // A) Location indicator
	EffectiveStart time.Time `gorm:"type:TIMESTAMP with time zone;index"`
	EffectiveEnd   *time.Time `gorm:"type:TIMESTAMP with time zone"`
	Schedule       string `gorm:"size:200"`      // D) زمان‌بندی اختیاری
	PlainText      string `gorm:"type:TEXT;not null"` // E) متن اصلی NOTAM
	LowerLimit     string `gorm:"size:20"`       // F) حد پایین ارتفاع
	UpperLimit     string `gorm:"size:20"`       // G) حد بالای ارتفاع

	// فیلدهای FAA/SWIM اضافی
	AirportName string `gorm:"size:150"`
	AffectedFIR string `gorm:"size:8;index"`
	IssuedAt    *time.Time `gorm:"type:TIMESTAMP with time zone"`

	// متن فرمت‌شده ICAO کامل (برای export به Jeppesen)
	FormattedText string `gorm:"type:TEXT"`

	// XML خام از FAA (برای audit و بازسازی)
	RawBody string `gorm:"type:TEXT"`

	// ---- تحلیل و امتیاز اهمیت (E3) ----
	QCode        string      `gorm:"size:8;index"`  // کد Q استخراج/دیکدشده مثل QMRLC
	QSubject     string      `gorm:"size:4"`        // موضوع (۲ حرف) MR
	QCondition   string      `gorm:"size:4"`        // وضعیت (۲ حرف) LC
	Traffic      string      `gorm:"size:4"`        // ترافیک هدف خط Q: IV/I/V (برای تطبیق IFR/VFR)
	Category     string      `gorm:"size:20;index"` // RUNWAY/NAVIGATION/AIRSPACE/…
	FlightPhases StringSlice `gorm:"type:TEXT"`     // DEPARTURE/ENROUTE/APPROACH/…
	Tags         StringSlice `gorm:"type:TEXT"`     // FICON/RWY_CLOSED/…
	BaseScore    int         `gorm:"index"`         // ۰..۱۰۰ مستقل از پرواز
	BaseLevel    string      `gorm:"size:10;index"` // CRITICAL/HIGH/MEDIUM/LOW/INFO
	WeightsVer   string      `gorm:"size:12"`       // نسخهٔ جدول وزن‌دهی (audit)

	// ---- هندسه و ارتفاع فضای هوایی (E5.6) — از خط Q؛ ستون area (PostGIS) از این‌ها ساخته می‌شود ----
	AreaLat      float64 // مرکز
	AreaLon      float64
	AreaRadiusNM float64 // شعاع NM؛ >۰ یعنی هندسه معلوم است
	LowerFt      int     // حد پایین ارتفاع (فوت)
	UpperFt      int     // حد بالا ارتفاع (فوت)
	VerticalKnown bool   // آیا حدود ارتفاعی معلوم است

	// ارجاع اختیاری (بدون FK - NOTAMها از فرودگاه‌های مختلف FAA می‌آیند)
	AirportICAO string `gorm:"size:8;index"` // کد ICAO محل - لزوماً در جدول airports نیست
	RunwayID    *uint  `gorm:"index"`
}

func (Notam) TableName() string { return "notams" }
