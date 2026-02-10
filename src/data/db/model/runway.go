package model

type Runway struct {
	BaseModel

	AirportICAO string `gorm:"size:4;index"`
	Name        string `gorm:"size:10"` // 13R/31L

	LEIdent string
	HEIdent string



	Notams []Notam `gorm:"foreignKey:RunwayID"`
}
