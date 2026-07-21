package webui_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quii/ce/internal/adapters/apiclient"
	"github.com/quii/ce/internal/adapters/webui"
	"github.com/quii/ce/internal/assert"
)

func TestIndex_ServesAPageThatFetchesTheGreetingOnLoadAndOnEveryKeystroke(t *testing.T) {
	handler := newTestHandler(t, fakeGreetingAPI(t, "Hello, World!"))

	rec := do(handler, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, rec.Code, http.StatusOK, "status")
	assert.Equal(t, rec.Header().Get("Content-Type"), "text/html; charset=utf-8", "Content-Type")

	body := rec.Body.String()
	for _, want := range []string{
		`id="greeting-hero"`,
		`hx-get="/greeting"`,
		`hx-target="#greeting-hero"`,
		`hx-trigger="keyup changed delay:300ms, load"`,
	} {
		assert.True(t, strings.Contains(body, want), "index page does not contain %q:\n%s", want, body)
	}
}

func TestGreeting_RendersTheGreetingFromTheAPI(t *testing.T) {
	handler := newTestHandler(t, fakeGreetingAPI(t, "Hello, Denise!"))

	rec := do(handler, httptest.NewRequest(http.MethodGet, "/greeting?name=Denise", nil))

	assert.Equal(t, rec.Code, http.StatusOK, "status")
	assert.Equal(t, strings.TrimSpace(rec.Body.String()), "Hello, Denise!", "body")
}

func TestGreeting_EscapesTheGreetingToPreventScriptInjection(t *testing.T) {
	handler := newTestHandler(t, fakeGreetingAPI(t, `<script>alert(1)</script>`))

	rec := do(handler, httptest.NewRequest(http.MethodGet, "/greeting", nil))

	got := rec.Body.String()
	assert.False(t, strings.Contains(got, "<script>"), "body contains unescaped %q:\n%s", "<script>", got)
}

func TestGreeting_RespondsWithBadGatewayWhenTheAPIFails(t *testing.T) {
	handler := newTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	rec := do(handler, httptest.NewRequest(http.MethodGet, "/greeting", nil))

	assert.Equal(t, rec.Code, http.StatusBadGateway, "status")
}

func do(handler *webui.Handler, req *http.Request) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	handler.Routes().ServeHTTP(rec, req)
	return rec
}

func newTestHandler(t *testing.T, apiHandler http.Handler) *webui.Handler {
	t.Helper()

	apiServer := httptest.NewServer(apiHandler)
	t.Cleanup(apiServer.Close)

	client, err := apiclient.NewClientWithResponses(apiServer.URL)
	assert.NoErr(t, err, "create api client")

	return webui.NewHandler(client)
}

func fakeGreetingAPI(t *testing.T, greeting string) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		assert.NoErr(t, json.NewEncoder(w).Encode(apiclient.Greeting{Greeting: greeting}), "encode fake greeting response")
	})
}
