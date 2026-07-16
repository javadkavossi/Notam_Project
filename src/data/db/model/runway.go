package model

type Runway struct {
	BaseModel

	AirportICAO string `gorm:"size:4;index"`
	Name        string `gorm:"size:16"` // 13R/31L

	LEIdent string `gorm:"size:8"` // شناسهٔ سر پایین باند (مثل 13R)
	HEIdent string `gorm:"size:8"` // شناسهٔ سر بالای باند (مثل 31L)

	LengthFt int
	WidthFt  int
	Surface  string `gorm:"size:40"`
	Lighted  bool
	Closed   bool

	Notams []Notam `gorm:"foreignKey:RunwayID"`
}
