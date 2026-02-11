package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/handlers"
	"github.com/hossein-repo/BaseProject/api/middleware"
)

// Notam ثبت روت‌های NOTAM
func Notam(g *gin.RouterGroup) {
	h := handlers.NewNotamHandler()
	g.GET("", h.List)
	g.GET("/alert-options", h.AlertOptions)
	g.GET("/alert-settings", middleware.NotamAuth, h.GetAlertSettings)
	g.PUT("/alert-settings", middleware.NotamAuth, h.SaveAlertSettings)
	g.GET("/recent", middleware.NotamAuth, h.Recent)
	g.GET("/by-series", h.GetBySeries)
	g.GET("/:id", h.GetByID)
}
