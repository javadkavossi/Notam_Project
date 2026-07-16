package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/handlers"
)

// Reference روت‌های دادهٔ مرجع (E7-5).
func Reference(g *gin.RouterGroup) {
	h := handlers.NewReferenceHandler()
	g.GET("/airports", h.AirportSearch)
	g.GET("/airports/:icao", h.AirportByICAO)
}
