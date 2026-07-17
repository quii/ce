package docs_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quii/ce/internal/adapters/docs"
)

func TestSpecHandler_ServesTheGivenSpec(t *testing.T) {
	spec := []byte("openapi: 3.0.3")
	handler := docs.SpecHandler(spec)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if got, want := rec.Header().Get("Content-Type"), "application/yaml"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	got, err := io.ReadAll(rec.Body)
	if err != nil {
		t.Fatalf("could not read response body: %v", err)
	}
	if string(got) != string(spec) {
		t.Errorf("body = %q, want %q", got, spec)
	}
}

func TestHandler_ServesAnHTMLPageReferencingTheSpec(t *testing.T) {
	handler := docs.Handler()

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if got, want := rec.Header().Get("Content-Type"), "text/html; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	got := rec.Body.String()
	if want := "/openapi.yaml"; !strings.Contains(got, want) {
		t.Errorf("docs page body does not reference %q:\n%s", want, got)
	}
}
