package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/quii/ce/api"
	"github.com/quii/ce/internal/adapters/docs"
	"github.com/quii/ce/internal/adapters/httpapi"
)

func main() {
	outPorts := NewOutPorts()
	app := NewApplication(outPorts)

	handler := httpapi.NewGreetingHandler(app.GetGreeting)
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
