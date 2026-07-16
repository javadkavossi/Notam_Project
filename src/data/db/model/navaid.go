package model

// Navaid کمک‌ناوبری (VOR/DME/NDB/TACAN/…) — برای تطبیق NOTAMهای ناوبری (E7).
type Navaid struct {
	BaseModel
	Ident               string  `gorm:"size:8;index"`  // شناسه مثل TEH
	Name                string  `gorm:"size:120"`
	Type                string  `gorm:"size:20;index"` // VOR/VOR-DME/DME/NDB/TACAN
	Frequency           string  `gorm:"size:20"`
	Lat                 float64
	Lon                 float64
	AssociatedAirportICAO string `gorm:"size:4;index"`
}
