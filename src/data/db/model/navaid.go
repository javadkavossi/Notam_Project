package model

// Navaid کمک‌ناوبری (VOR/DME/NDB/TACAN/…) — برای تطبیق NOTAMهای ناوبری (E7).
type Navaid struct {
	BaseModel
	Ident     string `gorm:"size:16;index"` // شناسه مثل TEH
	Name      string `gorm:"size:150"`
	Type      string `gorm:"size:30;index"` // VOR/VOR-DME/DME/NDB/TACAN
	Frequency string `gorm:"size:20"`
	Lat       float64
	Lon       float64
	// در دادهٔ منبع همیشه کد ICAO ۴حرفی نیست (شناسه‌های محلی بلندتر هم هست)
	AssociatedAirportICAO string `gorm:"size:16;index"`
}
