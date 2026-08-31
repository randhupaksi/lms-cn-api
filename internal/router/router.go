package router

import (
	"net/http"

	"ranvex-api/internal/config"
	"ranvex-api/internal/middleware"
	"ranvex-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type Engine struct{ *gin.Engine }

func New(cfg *config.Config) (*Engine, error) {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery(), middleware.RequestID())

	api := engine.Group(cfg.APIPrefix)
	api.GET("/health", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "Ranvex API is healthy", gin.H{"service": cfg.AppName, "environment": cfg.AppEnv})
	})

	return &Engine{Engine: engine}, nil
}
