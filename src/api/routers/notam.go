package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/handlers"
	"github.com/hossein-repo/BaseProject/api/middleware"
	"github.com/hossein-repo/BaseProject/config"
)

// Notam ثبت روت‌های NOTAM
func Notam(g *gin.RouterGroup, cfg *config.Config) {
	h := handlers.NewNotamHandler()
	auth := middleware.JWTAuth(cfg)

	g.GET("", h.List)
	g.GET("/alert-options", h.AlertOptions)
	g.GET("/alert-settings", auth, h.GetAlertSettings)
	g.PUT("/alert-settings", auth, h.SaveAlertSettings)
	g.GET("/recent", auth, h.Recent)
	g.GET("/by-series", h.GetBySeries)
	g.GET("/:id", h.GetByID)
}
