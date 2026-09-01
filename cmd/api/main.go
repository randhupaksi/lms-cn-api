package main

import (
	"log/slog"
	"os"

	"lms-cn-api/internal/app"
)

func main() {
	server, err := app.New()
	if err != nil {
		slog.Error("failed to bootstrap Citra Negara LMS API", "error", err)
		os.Exit(1)
	}
	if err := server.Run(); err != nil {
		slog.Error("failed to run Citra Negara LMS API", "error", err)
		os.Exit(1)
	}
}
