package main

import (
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/quii/ce/api"
	"github.com/quii/ce/internal/adapters/docs"
	"github.com/quii/ce/internal/adapters/httpapi"
	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/ports/in"
)

func main() {
	role := os.Getenv("CE_ROLE")
	if role == "" {
		role = "api"
	}

	switch role {
	case "api":
		runAPI()
	case "relay":
		runRelay()
	default:
		slog.Error("unknown role", "role", role)
		os.Exit(1)
	}
}

func runAPI() {
	greetings := memory.NewGreetingFinder()
	useCase := in.NewGetGreetingUseCase(greetings)
	handler := httpapi.NewGreetingHandler(useCase)
	strictHandler := httpapi.NewStrictHandler(handler, nil)

	mux := http.NewServeMux()
	httpapi.HandlerFromMux(strictHandler, mux)
	mux.Handle("GET /openapi.yaml", docs.SpecHandler(api.OpenAPISpec))
	mux.Handle("GET /docs", docs.Handler())

	slog.Info("starting api role", "addr", ":8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("api role stopped", "err", err)
		os.Exit(1)
	}
}

func runRelay() {
	slog.Info("starting relay role - nothing to drain yet")

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	slog.Info("relay role shutting down")
}
