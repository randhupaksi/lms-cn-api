package app

import (
	"lms-cn-api/internal/config"
	"lms-cn-api/internal/router"
)

type Server struct {
	engine *router.Engine
	port   string
}

func New() (*Server, error) {
	cfg := config.Load()
	engine, err := router.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Server{engine: engine, port: cfg.Port}, nil
}

func (s *Server) Run() error {
	return s.engine.Run(":" + s.port)
}
