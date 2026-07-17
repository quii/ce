package main

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/quii/ce/internal/adapters/apiclient"
	"github.com/quii/ce/internal/adapters/webui"
)

func main() {
	apiBaseURL := os.Getenv("API_BASE_URL")
	if apiBaseURL == "" {
		apiBaseURL = "http://localhost:8080"
	}

	client, err := apiclient.NewClientWithResponses(apiBaseURL)
	if err != nil {
		slog.Error("could not create api client", "err", err)
		os.Exit(1)
	}

	handler := webui.NewHandler(client)

	slog.Info("starting web", "addr", ":8080", "api_base_url", apiBaseURL)
	if err := http.ListenAndServe(":8080", handler.Routes()); err != nil {
		slog.Error("web stopped", "err", err)
		os.Exit(1)
	}
}
