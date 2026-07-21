package httpapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/quii/ce/internal/adapters/httpapi"
	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/ports/in"
)

func TestGreetingHandler_GetGreeting(t *testing.T) {
	useCase := in.NewGetGreetingUseCase(memory.NewGreetingFinder())
	handler := httpapi.NewGreetingHandler(useCase)

	name := "Chris"
	got, err := handler.GetGreeting(context.Background(), httpapi.GetGreetingRequestObject{
		Params: httpapi.GetGreetingParams{Name: &name},
	})
	assert.NoErr(t, err, "GetGreeting(%q)", name)

	want := httpapi.GetGreeting200JSONResponse{Greeting: "Hello, Chris!"}
	assert.Equal[httpapi.GetGreetingResponseObject](t, got, want, "GetGreeting(%q)", name)
}

func TestGreetingHandler_RepeatedNameParameterIsRejected(t *testing.T) {
	useCase := in.NewGetGreetingUseCase(memory.NewGreetingFinder())
	handler := httpapi.NewCompositeHandler(httpapi.NewGreetingHandler(useCase), conversationHandler(t))
	server := httpapi.Handler(httpapi.NewStrictHandler(handler, nil))

	req := httptest.NewRequest(http.MethodGet, "/greeting?name=Chris&name=Sam", nil)
	rec := httptest.NewRecorder()

	server.ServeHTTP(rec, req)

	assert.Equal(t, rec.Code, http.StatusBadRequest, "status")
}
