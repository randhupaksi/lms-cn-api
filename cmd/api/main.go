package main

import (
	"log/slog"
	"os"

	"ranvex-api/internal/app"
)

func main() {
	server, err := app.New()
	if err != nil {
		slog.Error("failed to bootstrap Ranvex API", "error", err)
		os.Exit(1)
	}
	if err := server.Run(); err != nil {
		slog.Error("failed to run Ranvex API", "error", err)
		os.Exit(1)
	}
}
