package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/opencode-ai/opencode-dog/internal/config"
	"github.com/opencode-ai/opencode-dog/internal/server"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	srv, err := server.New(cfg, logger)
	if err != nil {
		logger.Error("failed to create server", "error", err)
		os.Exit(1)
	}

	logger.Info("running database migrations")
	if err := srv.RunMigrations(context.Background(), "/app/migrations"); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migrations completed")

	srv.SyncOpencodeConfig(context.Background())

	serverURL := os.Getenv("OPENCODE_SERVER_URL")
	if serverURL != "" {
		if err := waitForOpencodeServer(logger, serverURL); err != nil {
			logger.Warn("opencode server not reachable, analyses may fail", "error", err)
		}
	}

	if err := srv.Start(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}

func waitForOpencodeServer(logger *slog.Logger, serverURL string) error {
	healthURL := serverURL + "/global/health"
	logger.Info("waiting for opencode server", "url", healthURL)

	deadline := time.After(60 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-deadline:
			return fmt.Errorf("timeout waiting for opencode server at %s", healthURL)
		case <-ticker.C:
			resp, err := http.Get(healthURL)
			if err == nil && resp.StatusCode == http.StatusOK {
				resp.Body.Close()
				logger.Info("opencode server is healthy")
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}
