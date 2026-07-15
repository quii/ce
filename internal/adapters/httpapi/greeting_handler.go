package httpapi

import (
	"encoding/json"
	"net/http"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

type GreetingHandler struct {
	useCase *in.GetGreetingUseCase
}

func NewGreetingHandler(useCase *in.GetGreetingUseCase) *GreetingHandler {
	return &GreetingHandler{useCase: useCase}
}

func (h *GreetingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := domain.NewName(r.URL.Query().Get("name"))
	greeting, err := h.useCase.Handle(r.Context(), in.GetGreetingCommand{Name: name})
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(greetingResponse{Greeting: string(greeting)})
}

type greetingResponse struct {
	Greeting string `json:"greeting"`
}
