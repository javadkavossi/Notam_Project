// Package token تولید و اعتبارسنجی JWT برای احراز هویت (E0-2).
package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt"
)

// Claims ادعاهای توکن: نام کاربری و نقش + فیلدهای استاندارد (انقضا و…).
type Claims struct {
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.StandardClaims
}

// Service تولید/پارس توکن با secret و مدت انقضا.
type Service struct {
	secret        []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

// NewService یک سرویس توکن می‌سازد.
func NewService(secret, refreshSecret string, accessTTL, refreshTTL time.Duration) *Service {
	return &Service{
		secret:        []byte(secret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token expired")
)

// GenerateAccess توکن دسترسی کوتاه‌عمر می‌سازد.
func (s *Service) GenerateAccess(username, role string) (string, error) {
	return s.sign(username, role, s.secret, s.accessTTL)
}

// GenerateRefresh توکن تازه‌سازی بلندعمر می‌سازد.
func (s *Service) GenerateRefresh(username, role string) (string, error) {
	return s.sign(username, role, s.refreshSecret, s.refreshTTL)
}

func (s *Service) sign(username, role string, key []byte, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := Claims{
		Username: username,
		Role:     role,
		StandardClaims: jwt.StandardClaims{
			IssuedAt:  now.Unix(),
			ExpiresAt: now.Add(ttl).Unix(),
			Subject:   username,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return tok.SignedString(key)
}

// ParseAccess توکن دسترسی را اعتبارسنجی و claims را برمی‌گرداند.
func (s *Service) ParseAccess(tokenStr string) (*Claims, error) {
	return parse(tokenStr, s.secret)
}

// ParseRefresh توکن تازه‌سازی را اعتبارسنجی می‌کند.
func (s *Service) ParseRefresh(tokenStr string) (*Claims, error) {
	return parse(tokenStr, s.refreshSecret)
}

func parse(tokenStr string, key []byte) (*Claims, error) {
	claims := &Claims{}
	tok, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		// فقط HMAC مجاز است (جلوگیری از حملهٔ alg=none / تعویض الگوریتم)
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return key, nil
	})
	if err != nil {
		if ve, ok := err.(*jwt.ValidationError); ok && ve.Errors&jwt.ValidationErrorExpired != 0 {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !tok.Valid || claims.Username == "" {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
