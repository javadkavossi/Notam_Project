package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/handlers"
	"github.com/hossein-repo/BaseProject/config"
)

func Auth(g *gin.RouterGroup, cfg *config.Config) {
	h := handlers.NewAuthHandler(cfg)
	g.POST("/login", h.Login)
	g.POST("/refresh", h.Refresh)
}
