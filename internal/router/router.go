package router

import (
	"net/http"

	"lms-cn-api/internal/config"
	"lms-cn-api/internal/middleware"
	"lms-cn-api/pkg/response"

	"github.com/gin-gonic/gin"
)

type Engine struct{ *gin.Engine }

func New(cfg *config.Config) (*Engine, error) {
	engine := gin.New()
	engine.Use(gin.Logger(), gin.Recovery(), middleware.RequestID())

	api := engine.Group(cfg.APIPrefix)
	api.GET("/health", func(c *gin.Context) {
		response.Success(c, http.StatusOK, "Citra Negara LMS API is healthy", gin.H{"service": cfg.AppName, "environment": cfg.AppEnv})
	})

	return &Engine{Engine: engine}, nil
}
