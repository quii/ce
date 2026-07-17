// Package webui is a thin HTML/HTMX view over the ce API - it holds no
// domain logic, only rendering, and talks to the API exclusively through
// the generated apiclient.
package webui

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/quii/ce/internal/adapters/apiclient"
)

// templates/*.gohtml are parsed once at package init - a new page is a
// new file dropped in that directory, not a change here. pico.css and
// htmx are loaded from a CDN rather than vendored, so this stays static
// files with no new Go dependency (same reasoning as internal/adapters/docs'
// Scalar page).
//
//go:embed templates/*.gohtml
var templateFS embed.FS

var templates = template.Must(template.ParseFS(templateFS, "templates/*.gohtml"))

type Handler struct {
	client *apiclient.ClientWithResponses
}

func NewHandler(client *apiclient.ClientWithResponses) *Handler {
	return &Handler{client: client}
}

func (h *Handler) Routes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", h.index)
	mux.HandleFunc("GET /greeting", h.greeting)
	return mux
}

func (h *Handler) index(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = templates.ExecuteTemplate(w, "index.gohtml", nil)
}

func (h *Handler) greeting(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")

	var params apiclient.GetGreetingParams
	if name != "" {
		params.Name = &name
	}

	resp, err := h.client.GetGreetingWithResponse(r.Context(), &params)
	if err != nil || resp.StatusCode() != http.StatusOK || resp.JSON200 == nil {
		http.Error(w, "could not fetch greeting", http.StatusBadGateway)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = templates.ExecuteTemplate(w, "greeting.gohtml", resp.JSON200.Greeting)
}
