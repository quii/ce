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
	projection := memory.NewProjection()
	starter := in.NewStartConversationUseCase(in.StartConversationDependencies{
		IDs:    memory.NewIDGenerator(),
		Clock:  memory.NewClock(),
		Events: events,
	})
	replier := in.NewReplyToThreadUseCase(in.ReplyToThreadDependencies{
		IDs:        memory.NewIDGenerator(),
		Clock:      memory.NewClock(),
		Events:     events,
		Projection: projection,
	})
	getter := in.NewGetConversationUseCase(projection)

	return httpapi.NewConversationHandler(starter, replier, getter)
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

func TestConversationHandler_ReplyToThread_MissingAuthorIsRejected(t *testing.T) {
	handler := conversationHandler(t)

	got, err := handler.ReplyToThread(context.Background(), httpapi.ReplyToThreadRequestObject{
		ConversationId: "conversation-1",
		ThreadId:       "thread-1",
		Body: &httpapi.ReplyToThreadJSONRequestBody{
			Text: strPtr("Let me know when you can"),
		},
	})
	if err != nil {
		t.Fatalf("ReplyToThread returned an unexpected transport error: %v", err)
	}

	if _, ok := got.(httpapi.ReplyToThread400JSONResponse); !ok {
		t.Errorf("ReplyToThread with no author = %#v, want a ReplyToThread400JSONResponse", got)
	}
}

func TestConversationHandler_ReplyToThread_UnknownConversationIs404(t *testing.T) {
	handler := conversationHandler(t)

	got, err := handler.ReplyToThread(context.Background(), httpapi.ReplyToThreadRequestObject{
		ConversationId: "does-not-exist",
		ThreadId:       "thread-1",
		Body: &httpapi.ReplyToThreadJSONRequestBody{
			Author: strPtr("user-1"),
			Text:   strPtr("Let me know when you can"),
		},
	})
	if err != nil {
		t.Fatalf("ReplyToThread returned an unexpected transport error: %v", err)
	}

	if _, ok := got.(httpapi.ReplyToThread404JSONResponse); !ok {
		t.Errorf("ReplyToThread(%q) = %#v, want a ReplyToThread404JSONResponse", "does-not-exist", got)
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
