package model

import "time"

// NotamAlertDelivery ثبت اعلان تحویل‌شده به کاربر (وقتی NOTAM با تنظیمات کاربر مطابقت داشت)
type NotamAlertDelivery struct {
	Id        int       `gorm:"primaryKey"`
	Username  string    `gorm:"size:120;index:idx_alert_delivery_user_created;not null"`
	NotamId   int       `gorm:"index:idx_alert_delivery_user_created;not null"`
	CreatedAt time.Time `gorm:"type:TIMESTAMP with time zone;not null"`
}

func (NotamAlertDelivery) TableName() string { return "notam_alert_deliveries" }
