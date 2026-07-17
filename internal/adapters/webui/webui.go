// Package webui is a thin HTML/HTMX view over the ce API - it holds no
// domain logic, only rendering, and talks to the API exclusively through
// the generated apiclient.
package webui

import (
	"html/template"
	"net/http"

	"github.com/quii/ce/internal/adapters/apiclient"
)

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
	_ = indexTemplate.Execute(w, nil)
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
	_ = greetingTemplate.Execute(w, resp.JSON200.Greeting)
}

var indexTemplate = template.Must(template.New("index").Parse(`<!doctype html>
<html>
<head>
  <title>ce</title>
  <script src="https://unpkg.com/htmx.org@2"></script>
</head>
<body>
  <h1>ce</h1>
  <form hx-get="/greeting" hx-target="#greeting-result">
    <label>Name <input type="text" name="name"></label>
    <button type="submit">Greet me</button>
  </form>
  <div id="greeting-result"></div>
</body>
</html>
`))

var greetingTemplate = template.Must(template.New("greeting").Parse(`<p>{{.}}</p>`))
