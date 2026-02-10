package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// StringSlice برای ذخیره آرایهٔ رشته در دیتابیس به صورت JSON
type StringSlice []string

func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}

func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}
	var b []byte
	switch v := value.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return errors.New("invalid type for StringSlice")
	}
	return json.Unmarshal(b, s)
}

// NotamAlertSettings تنظیمات اعلان NOTAM هر کاربر (بر اساس نام کاربری)
type NotamAlertSettings struct {
	BaseModel
	Username         string      `gorm:"size:120;uniqueIndex;not null"`
	SelectedFirs     StringSlice `gorm:"type:TEXT"`
	SelectedAirports StringSlice `gorm:"type:TEXT"`
	SelectedKeywords StringSlice `gorm:"type:TEXT"`
	CustomKeywords   StringSlice `gorm:"type:TEXT"` // کلیدواژه‌های اضافه‌شده توسط کاربر
}

func (NotamAlertSettings) TableName() string { return "notam_alert_settings" }
