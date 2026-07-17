package webui_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quii/ce/internal/adapters/apiclient"
	"github.com/quii/ce/internal/adapters/webui"
)

func TestIndex_ServesAPageThatFetchesTheGreetingOnLoadAndOnEveryKeystroke(t *testing.T) {
	handler := newTestHandler(t, fakeGreetingAPI(t, "Hello, World!"))

	rec := do(handler, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	body := rec.Body.String()
	for _, want := range []string{
		`id="greeting-hero"`,
		`hx-get="/greeting"`,
		`hx-target="#greeting-hero"`,
		`hx-trigger="keyup changed delay:300ms, load"`,
	} {
		if !strings.Contains(body, want) {
			t.Errorf("index page does not contain %q:\n%s", want, body)
		}
	}
}

func TestGreeting_RendersTheGreetingFromTheAPI(t *testing.T) {
	handler := newTestHandler(t, fakeGreetingAPI(t, "Hello, Denise!"))

	rec := do(handler, httptest.NewRequest(http.MethodGet, "/greeting?name=Denise", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := strings.TrimSpace(rec.Body.String()), "Hello, Denise!"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
}

func TestGreeting_EscapesTheGreetingToPreventScriptInjection(t *testing.T) {
	handler := newTestHandler(t, fakeGreetingAPI(t, `<script>alert(1)</script>`))

	rec := do(handler, httptest.NewRequest(http.MethodGet, "/greeting", nil))

	if got, dontWant := rec.Body.String(), "<script>"; strings.Contains(got, dontWant) {
		t.Errorf("body contains unescaped %q:\n%s", dontWant, got)
	}
}

func TestGreeting_RespondsWithBadGatewayWhenTheAPIFails(t *testing.T) {
	handler := newTestHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	rec := do(handler, httptest.NewRequest(http.MethodGet, "/greeting", nil))

	if rec.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadGateway)
	}
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
	if err != nil {
		t.Fatalf("could not create api client: %v", err)
	}

	return webui.NewHandler(client)
}

func fakeGreetingAPI(t *testing.T, greeting string) http.Handler {
	t.Helper()

	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(apiclient.Greeting{Greeting: greeting}); err != nil {
			t.Fatalf("could not encode fake greeting response: %v", err)
		}
	})
}
