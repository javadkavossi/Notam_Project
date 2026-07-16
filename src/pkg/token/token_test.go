package token

import (
	"testing"
	"time"
)

func newSvc(accessTTL time.Duration) *Service {
	return NewService("secret-a", "secret-r", accessTTL, time.Hour)
}

func TestGenerateAndParseAccess(t *testing.T) {
	s := newSvc(time.Hour)
	tok, err := s.GenerateAccess("alice", "admin")
	if err != nil {
		t.Fatalf("GenerateAccess: %v", err)
	}
	claims, err := s.ParseAccess(tok)
	if err != nil {
		t.Fatalf("ParseAccess: %v", err)
	}
	if claims.Username != "alice" || claims.Role != "admin" {
		t.Errorf("claims نادرست: %+v", claims)
	}
}

func TestExpiredToken(t *testing.T) {
	s := newSvc(-time.Minute) // منقضی‌شده
	tok, _ := s.GenerateAccess("bob", "viewer")
	if _, err := s.ParseAccess(tok); err != ErrExpiredToken {
		t.Errorf("انتظار ErrExpiredToken، دریافت %v", err)
	}
}

func TestTamperedOrWrongSecret(t *testing.T) {
	s := newSvc(time.Hour)
	tok, _ := s.GenerateAccess("carol", "operator")

	// توکن دسترسی نباید با secret تازه‌سازی معتبر باشد
	if _, err := s.ParseRefresh(tok); err == nil {
		t.Error("توکن access نباید به‌عنوان refresh معتبر باشد")
	}
	// دستکاری امضا
	if _, err := s.ParseAccess(tok + "x"); err == nil {
		t.Error("توکن دستکاری‌شده نباید معتبر باشد")
	}
	// secret متفاوت
	other := NewService("different", "r", time.Hour, time.Hour)
	if _, err := other.ParseAccess(tok); err == nil {
		t.Error("توکن با secret متفاوت نباید معتبر باشد")
	}
}

func TestRefreshRoundTrip(t *testing.T) {
	s := newSvc(time.Hour)
	rt, err := s.GenerateRefresh("dave", "admin")
	if err != nil {
		t.Fatalf("GenerateRefresh: %v", err)
	}
	claims, err := s.ParseRefresh(rt)
	if err != nil || claims.Username != "dave" || claims.Role != "admin" {
		t.Fatalf("ParseRefresh نادرست: claims=%+v err=%v", claims, err)
	}
}
