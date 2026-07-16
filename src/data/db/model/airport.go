package model

type Airport struct {
	BaseModel
	ICAO        string  `gorm:"size:4;unique;not null;index"`
	IATA        string  `gorm:"size:4;index"`
	Name        string  `gorm:"size:150"`
	Country     string  `gorm:"size:2;index"` // ISO country
	Municipality string `gorm:"size:120"`
	Type        string  `gorm:"size:30"` // large_airport/medium_airport/...
	Lat         float64
	Lon         float64
	ElevationFt int

	Runways []Runway `gorm:"foreignKey:AirportICAO;references:ICAO"`
}
