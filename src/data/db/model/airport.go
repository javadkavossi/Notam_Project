package model

type Airport struct {
	BaseModel
	ICAO string `gorm:"size:4;unique;not null;index"`
	Name string `gorm:"size:100"`
	Lat  float64
	Lon  float64

	Runways []Runway `gorm:"foreignKey:AirportICAO;references:ICAO"`
}
