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

	// ارجاع اختیاری (بدون FK - NOTAMها از فرودگاه‌های مختلف FAA می‌آیند)
	AirportICAO string `gorm:"size:8;index"` // کد ICAO محل - لزوماً در جدول airports نیست
	RunwayID    *uint  `gorm:"index"`
}

func (Notam) TableName() string { return "notams" }
