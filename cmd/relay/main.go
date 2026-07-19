package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	databaseURL := mustEnv("DATABASE_URL")

	outPorts, err := NewOutPorts(ctx, databaseURL)
	if err != nil {
		slog.Error("failed to build out ports", "err", err)
		os.Exit(1)
	}
	app := NewApplication(outPorts)

	slog.Info("starting relay")

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("relay shutting down")
			return
		case <-ticker.C:
			if err := app.Drain.Drain(ctx); err != nil {
				slog.Error("relay drain failed", "err", err)
			}
		}
	}
}

func mustEnv(name string) string {
	value := os.Getenv(name)
	if value == "" {
		slog.Error("missing required environment variable", "name", name)
		os.Exit(1)
	}
	return value
}
