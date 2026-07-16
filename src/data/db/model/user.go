package model

// نقش‌های کاربر (E0-3)
const (
	RoleViewer   = "viewer"   // فقط مشاهده
	RoleOperator = "operator" // مشاهده + عملیات
	RoleAdmin    = "admin"    // مدیریت کامل
)

// User کاربر احراز هویت‌شده با رمز هش‌شده و نقش.
type User struct {
	BaseModel
	Username     string `gorm:"size:120;uniqueIndex;not null"`
	PasswordHash string `gorm:"size:200;not null"`
	Role         string `gorm:"size:20;not null;default:viewer"`
}

func (User) TableName() string { return "users" }
