package httpapi_test

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/adapters/httpapi"
	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/ports/in"
)

func conversationHandler(t *testing.T) *httpapi.ConversationHandler {
	t.Helper()

	events := memory.NewEventStore()
	starter := in.NewStartConversationUseCase(in.StartConversationDependencies{
		IDs:    memory.NewIDGenerator(),
		Clock:  memory.NewClock(),
		Events: events,
	})
	getter := in.NewGetConversationUseCase(memory.NewProjection())

	return httpapi.NewConversationHandler(starter, getter)
}

func strPtr(s string) *string { return &s }

func TestConversationHandler_StartConversation_MissingResourceURLIsRejected(t *testing.T) {
	handler := conversationHandler(t)

	got, err := handler.StartConversation(context.Background(), httpapi.StartConversationRequestObject{
		Body: &httpapi.StartConversationJSONRequestBody{
			ThreadTitle: strPtr("Order query"),
			Author:      strPtr("user-1"),
			Recipients:  &[]string{},
			Message:     strPtr("Where is my order?"),
		},
	})
	if err != nil {
		t.Fatalf("StartConversation returned an unexpected transport error: %v", err)
	}

	if _, ok := got.(httpapi.StartConversation400JSONResponse); !ok {
		t.Errorf("StartConversation with no resourceUrl = %#v, want a StartConversation400JSONResponse", got)
	}
}

func TestConversationHandler_GetConversation_UnknownIDIs404(t *testing.T) {
	handler := conversationHandler(t)

	got, err := handler.GetConversation(context.Background(), httpapi.GetConversationRequestObject{Id: "does-not-exist"})
	if err != nil {
		t.Fatalf("GetConversation returned an unexpected transport error: %v", err)
	}

	if _, ok := got.(httpapi.GetConversation404JSONResponse); !ok {
		t.Errorf("GetConversation(%q) = %#v, want a GetConversation404JSONResponse", "does-not-exist", got)
	}
}
