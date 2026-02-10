package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/helper"
)

const (
	BearerPrefix   = "Bearer "
	TokenPrefix    = "notam-token-"
	ContextUserKey = "username"
)

// NotamAuth استخراج نام کاربری از توکن و قرار دادن در context (برای روت‌های محافظت‌شده)
func NotamAuth(c *gin.Context) {
	auth := c.GetHeader("Authorization")
	if auth == "" || !strings.HasPrefix(auth, BearerPrefix) {
		c.JSON(http.StatusUnauthorized, helper.GenerateBaseResponse(nil, false, helper.AuthError))
		c.Abort()
		return
	}
	token := strings.TrimPrefix(auth, BearerPrefix)
	if token == "" || !strings.HasPrefix(token, TokenPrefix) {
		c.JSON(http.StatusUnauthorized, helper.GenerateBaseResponse(nil, false, helper.AuthError))
		c.Abort()
		return
	}
	username := strings.TrimPrefix(token, TokenPrefix)
	if username == "" {
		c.JSON(http.StatusUnauthorized, helper.GenerateBaseResponse(nil, false, helper.AuthError))
		c.Abort()
		return
	}
	c.Set(ContextUserKey, username)
	c.Next()
}

// GetUsername نام کاربری از context (بعد از NotamAuth)
func GetUsername(c *gin.Context) string {
	v, _ := c.Get(ContextUserKey)
	s, _ := v.(string)
	return s
}
