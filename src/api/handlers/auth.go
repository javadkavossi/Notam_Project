package handlers

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/helper"
)

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  string `json:"user"`
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, helper.GenerateBaseResponseWithError(nil, false, helper.ValidationError, err))
		return
	}
	user := os.Getenv("AUTH_USER")
	if user == "" {
		user = "admin"
	}
	pass := os.Getenv("AUTH_PASS")
	if pass == "" {
		pass = "admin"
	}
	if req.Username != user || req.Password != pass {
		c.JSON(http.StatusUnauthorized, helper.GenerateBaseResponse(nil, false, helper.AuthError))
		return
	}
	token := "notam-token-" + user
	c.JSON(http.StatusOK, helper.GenerateBaseResponse(LoginResponse{Token: token, User: user}, true, helper.Success))
}
