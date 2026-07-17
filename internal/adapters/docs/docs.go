// Package docs serves the OpenAPI contract and an interactive
// documentation UI for it. This is infrastructure, not a use-case-backed
// HTTP adapter - there's no request to parse or in-port to call, so it
// sits outside docs/adr/0007-thin-http-handlers.md's scope by design.
package docs

import "net/http"

func SpecHandler(spec []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(spec)
	})
}

// Handler serves a Scalar-based documentation UI pointed at the spec
// served by SpecHandler. Scalar itself is loaded from a CDN rather than
// vendored, so this stays a single static page with no new dependency.
func Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(page))
	})
}

const page = `<!doctype html>
<html>
<head><title>ce API docs</title></head>
<body>
  <div id="app"></div>
  <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  <script>Scalar.createApiReference('#app', { url: '/openapi.yaml' })</script>
</body>
</html>
`
