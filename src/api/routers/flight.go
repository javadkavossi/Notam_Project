package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/handlers"
	"github.com/hossein-repo/BaseProject/api/middleware"
	"github.com/hossein-repo/BaseProject/config"
)

// Flight روت‌های پرواز و بریفینگ (E5). همه نیازمند احراز هویت.
func Flight(g *gin.RouterGroup, cfg *config.Config) {
	h := handlers.NewFlightHandler()
	auth := middleware.JWTAuth(cfg)

	g.POST("", auth, h.CreateFlight)
	g.GET("", auth, h.ListFlights)
	g.POST("/briefing", auth, h.PreviewBriefing)
	g.GET("/:id/briefing", auth, h.Briefing)
}
