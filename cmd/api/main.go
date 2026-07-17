package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/quii/ce/api"
	"github.com/quii/ce/internal/adapters/docs"
	"github.com/quii/ce/internal/adapters/httpapi"
	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/ports/in"
)

func main() {
	greetings := memory.NewGreetingFinder()
	useCase := in.NewGetGreetingUseCase(greetings)
	handler := httpapi.NewGreetingHandler(useCase)
	strictHandler := httpapi.NewStrictHandler(handler, nil)

	mux := http.NewServeMux()
	httpapi.HandlerFromMux(strictHandler, mux)
	mux.Handle("GET /openapi.yaml", docs.SpecHandler(api.OpenAPISpec))
	mux.Handle("GET /docs", docs.Handler())

	slog.Info("starting api", "addr", ":8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		slog.Error("api stopped", "err", err)
		os.Exit(1)
	}
}
