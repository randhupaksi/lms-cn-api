package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"lms-cn-api/internal/config"
	"lms-cn-api/internal/database"
	"lms-cn-api/internal/router"
)

type Server struct {
	config     *config.Config
	connection *database.Connection
	engine     *router.Engine
}

func New() (*Server, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("load configuration: %w", err)
	}
	connection, err := database.Open(cfg)
	if err != nil {
		return nil, err
	}
	if err := database.RunMigrations(context.Background(), connection.GORM, cfg.MigrationsPath); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("run database migrations: %w", err)
	}
	if cfg.SeedDemoData {
		if err := database.SeedDemoData(context.Background(), connection.GORM); err != nil {
			_ = connection.Close()
			return nil, fmt.Errorf("seed demo data: %w", err)
		}
	}
	engine, err := router.New(cfg, connection.GORM)
	if err != nil {
		_ = connection.Close()
		return nil, err
	}
	return &Server{config: cfg, connection: connection, engine: engine}, nil
}

func (s *Server) Run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go s.engine.ExpiryWorker.Run(ctx)

	httpServer := &http.Server{Addr: ":" + s.config.Port, Handler: s.engine.Engine, ReadHeaderTimeout: s.config.ShutdownTimeout}
	errorsChannel := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errorsChannel <- err
		}
		close(errorsChannel)
	}()

	select {
	case err := <-errorsChannel:
		_ = s.connection.Close()
		return err
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()
	shutdownErr := httpServer.Shutdown(shutdownCtx)
	databaseErr := s.connection.Close()
	return errors.Join(shutdownErr, databaseErr)
}
