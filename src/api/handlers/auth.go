package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/helper"
	"github.com/hossein-repo/BaseProject/config"
	"github.com/hossein-repo/BaseProject/internal/storage"
	"github.com/hossein-repo/BaseProject/pkg/token"
)

// AuthHandler احراز هویت با JWT و کاربران دیتابیسی (E0-2/E0-3).
type AuthHandler struct {
	tokens *token.Service
	users  *storage.UserRepository
}

func NewAuthHandler(cfg *config.Config) *AuthHandler {
	return &AuthHandler{
		tokens: token.NewService(cfg.JWT.Secret, cfg.JWT.RefreshSecret,
			cfg.JWT.AccessTokenExpireDuration, cfg.JWT.RefreshTokenExpireDuration),
		users: storage.NewUserRepository(),
	}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token        string `json:"token"` // access token (سازگاری با فرانت فعلی)
	RefreshToken string `json:"refreshToken"`
	User         string `json:"user"`
	Role         string `json:"role"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// Login godoc
// @Summary ورود و دریافت توکن JWT
// @Tags auth
// @Accept json
// @Produce json
// @Param body body LoginRequest true "اعتبارنامه"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponseWithError(nil, false, helper.ValidationError, err))
		return
	}

	u, err := h.users.FindByUsername(req.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	// پیام یکسان برای «کاربر نیست» و «رمز غلط» تا نشت اطلاعات نداشته باشیم
	if u == nil || !storage.VerifyPassword(u.PasswordHash, req.Password) {
		c.JSON(http.StatusUnauthorized, helper.GenerateBaseResponse(nil, false, helper.AuthError))
		return
	}

	access, err := h.tokens.GenerateAccess(u.Username, u.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	refresh, err := h.tokens.GenerateRefresh(u.Username, u.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(LoginResponse{
		Token: access, RefreshToken: refresh, User: u.Username, Role: u.Role,
	}, true, helper.Success))
}

// Refresh godoc
// @Summary تازه‌سازی توکن دسترسی با refresh token
// @Tags auth
// @Accept json
// @Produce json
// @Param body body RefreshRequest true "refresh token"
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponseWithError(nil, false, helper.ValidationError, err))
		return
	}
	claims, err := h.tokens.ParseRefresh(req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, helper.GenerateBaseResponse(nil, false, helper.AuthError))
		return
	}
	access, err := h.tokens.GenerateAccess(claims.Username, claims.Role)
	if err != nil {
		c.JSON(http.StatusInternalServerError, helper.GenerateBaseResponseWithError(nil, false, helper.InternalError, err))
		return
	}
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(LoginResponse{
		Token: access, RefreshToken: req.RefreshToken, User: claims.Username, Role: claims.Role,
	}, true, helper.Success))
}
