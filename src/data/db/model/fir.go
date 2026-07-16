package model

// FIR منطقهٔ اطلاعات پروازی با مرز جغرافیایی — برای تطبیق NOTAMهای enroute (E7-3).
//
// مرز به دو شکل نگهداری می‌شود:
//   - BoundaryGeoJSON: متن GeoJSON (توسط GORM مدیریت می‌شود)
//   - ستون PostGIS `boundary geometry(Geometry,4326)`: از GeoJSON ساخته می‌شود (در migration/loader
//     با ST_GeomFromGeoJSON) و برای کوئری فضایی ST_Intersects استفاده می‌شود. این ستون در مدل GORM
//     نیست تا اسکن geometry مشکل ایجاد نکند.
type FIR struct {
	BaseModel
	ICAO           string `gorm:"size:4;uniqueIndex;not null"`
	Name           string `gorm:"size:150"`
	BoundaryGeoJSON string `gorm:"type:TEXT"`
}

func (FIR) TableName() string { return "firs" }

// RefDatasetVersion نسخه و checksum هر دیتاست مرجع برای نسخه‌بندی و تشخیص تغییر (E7-4).
type RefDatasetVersion struct {
	BaseModel
	Dataset   string `gorm:"size:40;index;not null"` // airports/runways/navaids/firs
	Checksum  string `gorm:"size:64"`                // sha256 منبع
	RowCount  int
	Note      string `gorm:"size:200"`
}

func (RefDatasetVersion) TableName() string { return "ref_dataset_versions" }
