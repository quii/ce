package httpapi_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quii/ce/internal/adapters/httpapi"
	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/ports/in"
)

func TestGreetingHandler_RepeatedNameParameterUsesFirstValue(t *testing.T) {
	useCase := in.NewGetGreetingUseCase(memory.NewGreetingFinder())
	handler := httpapi.NewGreetingHandler(useCase)

	req := httptest.NewRequest(http.MethodGet, "/greeting?name=Chris&name=Sam", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	var body struct {
		Greeting string `json:"greeting"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("could not decode response body: %v", err)
	}

	want := "Hello, Chris!"
	if body.Greeting != want {
		t.Errorf("greeting = %q, want %q", body.Greeting, want)
	}
}
