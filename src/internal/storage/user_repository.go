package storage

import (
	"errors"
	"log"
	"strings"

	"github.com/hossein-repo/BaseProject/data/db"
	"github.com/hossein-repo/BaseProject/data/db/model"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserRepository دسترسی به کاربران احراز هویت (E0-3).
type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository() *UserRepository {
	return &UserRepository{db: db.GetDb()}
}

// FindByUsername کاربر را با نام کاربری برمی‌گرداند (nil اگر نبود).
func (r *UserRepository) FindByUsername(username string) (*model.User, error) {
	var u model.User
	err := r.db.Where("username = ?", strings.TrimSpace(username)).First(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// VerifyPassword رمز خام را با هش ذخیره‌شده مقایسه می‌کند.
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// HashPassword رمز خام را هش می‌کند.
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

// EnsureUser اگر کاربری با این نام نباشد، آن را با رمز و نقش داده‌شده می‌سازد (seed اولیه).
// اگر از قبل باشد، دست نمی‌خورد (رمز بازنویسی نمی‌شود).
func (r *UserRepository) EnsureUser(username, plainPassword, role string) error {
	username = strings.TrimSpace(username)
	if username == "" || plainPassword == "" {
		return errors.New("username/password required for seed user")
	}
	existing, err := r.FindByUsername(username)
	if err != nil {
		return err
	}
	if existing != nil {
		return nil // از قبل هست
	}
	hash, err := HashPassword(plainPassword)
	if err != nil {
		return err
	}
	u := model.User{Username: username, PasswordHash: hash, Role: role}
	if err := r.db.Create(&u).Error; err != nil {
		return err
	}
	log.Printf("👤 Seeded user '%s' with role '%s'", username, role)
	return nil
}
