package api

import (
	"fmt"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"github.com/hossein-repo/BaseProject/api/middleware"
	"github.com/hossein-repo/BaseProject/api/routers"
	"github.com/hossein-repo/BaseProject/config"
	"github.com/hossein-repo/BaseProject/docs"
	"github.com/hossein-repo/BaseProject/pkg/logging"
)

func InitServer(cfg *config.Config) {
	r := gin.New()

	r.Use(middleware.DefaultStructuredLogger(cfg))
	r.Use(middleware.Cors(cfg))
	r.Use(gin.Logger(), gin.CustomRecovery(middleware.ErrorHandler))

	registerRoutes(r, cfg)
	registerSwagger(r, cfg)

	logger := logging.NewLogger(cfg)
	logger.Info(logging.General, logging.Startup, "API Server started", nil)
	_ = r.Run(fmt.Sprintf(":%s", cfg.Server.InternalPort))
}

func registerRoutes(r *gin.Engine, cfg *config.Config) {
	api := r.Group("/api")
	v1 := api.Group("/v1")
	{
		health := v1.Group("/health")
		routers.Health(health)
		auth := v1.Group("/auth")
		routers.Auth(auth, cfg)
		notams := v1.Group("/notams")
		routers.Notam(notams, cfg)
		ref := v1.Group("/reference")
		routers.Reference(ref)
		flights := v1.Group("/flights")
		routers.Flight(flights, cfg)
	}
}

func registerSwagger(r *gin.Engine, cfg *config.Config) {
	docs.SwaggerInfo.Title = "NOTAM API"
	docs.SwaggerInfo.Description = "FAA NOTAM Consumer API"
	docs.SwaggerInfo.Version = "1.0"
	docs.SwaggerInfo.BasePath = "/api"
	docs.SwaggerInfo.Host = fmt.Sprintf("localhost:%s", cfg.Server.ExternalPort)
	docs.SwaggerInfo.Schemes = []string{"http", "https"}

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
}
