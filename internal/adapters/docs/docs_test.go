package docs_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/quii/ce/internal/adapters/docs"
	"github.com/quii/ce/internal/assert"
)

func TestSpecHandler_ServesTheGivenSpec(t *testing.T) {
	spec := []byte("openapi: 3.0.3")
	handler := docs.SpecHandler(spec)

	req := httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, rec.Header().Get("Content-Type"), "application/yaml", "Content-Type")

	got, err := io.ReadAll(rec.Body)
	assert.NoErr(t, err, "read response body")
	assert.Equal(t, string(got), string(spec), "body")
}

func TestHandler_ServesAnHTMLPageReferencingTheSpec(t *testing.T) {
	handler := docs.Handler()

	req := httptest.NewRequest(http.MethodGet, "/docs", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusOK, "status")
	assert.Equal(t, rec.Header().Get("Content-Type"), "text/html; charset=utf-8", "Content-Type")

	got := rec.Body.String()
	assert.True(t, strings.Contains(got, "/openapi.yaml"), "docs page body does not reference %q:\n%s", "/openapi.yaml", got)
}
