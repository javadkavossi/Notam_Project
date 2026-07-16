package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/helper"
	"github.com/hossein-repo/BaseProject/config"
	"github.com/hossein-repo/BaseProject/pkg/token"
)

const (
	BearerPrefix   = "Bearer "
	ContextUserKey = "username"
	ContextRoleKey = "role"
)

// JWTAuth یک middleware می‌سازد که توکن JWT را اعتبارسنجی و username/role را در context قرار می‌دهد (E0-2).
func JWTAuth(cfg *config.Config) gin.HandlerFunc {
	svc := token.NewService(cfg.JWT.Secret, cfg.JWT.RefreshSecret,
		cfg.JWT.AccessTokenExpireDuration, cfg.JWT.RefreshTokenExpireDuration)

	return func(c *gin.Context) {
		auth := c.GetHeader("Authorization")
		if !strings.HasPrefix(auth, BearerPrefix) {
			unauthorized(c)
			return
		}
		tokenStr := strings.TrimSpace(strings.TrimPrefix(auth, BearerPrefix))
		claims, err := svc.ParseAccess(tokenStr)
		if err != nil {
			unauthorized(c)
			return
		}
		c.Set(ContextUserKey, claims.Username)
		c.Set(ContextRoleKey, claims.Role)
		c.Next()
	}
}

// RequireRole دسترسی را به نقش‌های مجاز محدود می‌کند (بعد از JWTAuth استفاده شود).
func RequireRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}
	return func(c *gin.Context) {
		if !allowed[GetRole(c)] {
			c.JSON(http.StatusForbidden, helper.GenerateBaseResponse(nil, false, helper.ForbiddenError))
			c.Abort()
			return
		}
		c.Next()
	}
}

func unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, helper.GenerateBaseResponse(nil, false, helper.AuthError))
	c.Abort()
}

// GetUsername نام کاربری از context (بعد از JWTAuth).
func GetUsername(c *gin.Context) string {
	v, _ := c.Get(ContextUserKey)
	s, _ := v.(string)
	return s
}

// GetRole نقش کاربر از context (بعد از JWTAuth).
func GetRole(c *gin.Context) string {
	v, _ := c.Get(ContextRoleKey)
	s, _ := v.(string)
	return s
}
