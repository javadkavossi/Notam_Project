package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/helper"
)

// HealthHandler handles health check
type HealthHandler struct{}

// NewHealthHandler returns a new HealthHandler
func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

// Health godoc
// @Summary Health check
// @Description Check API status
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} helper.BaseHttpResponse
// @Router /api/v1/health/ [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, helper.GenerateBaseResponse("OK", true, helper.Success))
}
