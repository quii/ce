package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quii/ce/internal/adapters/httpapi"
	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/ports/in"
)

func TestGreetingHandler_GetGreeting(t *testing.T) {
	useCase := in.NewGetGreetingUseCase(memory.NewGreetingFinder())
	handler := httpapi.NewGreetingHandler(useCase)

	name := "Chris"
	got, err := handler.GetGreeting(context.Background(), httpapi.GetGreetingRequestObject{
		Params: httpapi.GetGreetingParams{Name: &name},
	})
	if err != nil {
		t.Fatalf("GetGreeting(%q) returned an unexpected error: %v", name, err)
	}

	want := httpapi.GetGreeting200JSONResponse{Greeting: "Hello, Chris!"}
	if got != want {
		t.Errorf("GetGreeting(%q) = %#v, want %#v", name, got, want)
	}
}

func TestGreetingHandler_RepeatedNameParameterIsRejected(t *testing.T) {
	useCase := in.NewGetGreetingUseCase(memory.NewGreetingFinder())
	handler := httpapi.NewGreetingHandler(useCase)
	server := httpapi.Handler(httpapi.NewStrictHandler(handler, nil))

	req := httptest.NewRequest(http.MethodGet, "/greeting?name=Chris&name=Sam", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}
