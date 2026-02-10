package routers

import (
	"github.com/gin-gonic/gin"
	"github.com/hossein-repo/BaseProject/api/handlers"
)

func Auth(g *gin.RouterGroup) {
	g.POST("/login", handlers.Login)
}
