package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"

	"github.com/quii/ce/api"
	"github.com/quii/ce/internal/adapters/docs"
	"github.com/quii/ce/internal/adapters/httpapi"
	"github.com/quii/ce/internal/adapters/postgres"
)

func main() {
	ctx := context.Background()

	databaseURL := mustEnv("DATABASE_URL")

	if err := postgres.Migrate(ctx, databaseURL); err != nil {
		slog.Error("failed to apply migrations", "err", err)
		os.Exit(1)
	}

	outPorts, err := NewOutPorts(ctx, databaseURL)
	if err != nil {
		slog.Error("failed to build out ports", "err", err)
		os.Exit(1)
	}
	app := NewApplication(outPorts)

	handler := httpapi.NewCompositeHandler(
		httpapi.NewGreetingHandler(app.GetGreeting),
		httpapi.NewConversationHandler(app.StartConversation, app.AddThread, app.ReplyToThread, app.ManageThreadParticipants, app.GetConversation, app.ListConversationEvents, app.GetConversationsByParticipant),
	)
	strictHandler := httpapi.NewStrictHandler(handler, nil)

	apiMux := http.NewServeMux()
	httpapi.HandlerFromMux(strictHandler, apiMux)

	mux := http.NewServeMux()
	mux.Handle("/", apiMux)
	mux.Handle("GET /openapi.yaml", docs.SpecHandler(api.OpenAPISpec))
	mux.Handle("GET /docs", docs.Handler())

	slog.Info("starting api", "addr", ":8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("api stopped", "err", err)
		os.Exit(1)
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
