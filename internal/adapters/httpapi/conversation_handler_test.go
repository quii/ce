package httpapi_test

import (
	"context"
	"testing"

	"github.com/quii/ce/internal/adapters/httpapi"
	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/ports/in"
)

func conversationHandler(t *testing.T) *httpapi.ConversationHandler {
	t.Helper()

	events := memory.NewEventStore()
	projection := memory.NewProjection()
	starter := in.NewStartConversationUseCase(memory.NewIDGenerator(), memory.NewClock(), events)
	adder := in.NewAddThreadUseCase(memory.NewIDGenerator(), memory.NewClock(), events, projection)
	replier := in.NewReplyToThreadUseCase(memory.NewIDGenerator(), memory.NewClock(), events, projection)
	participants := in.NewManageThreadParticipantUseCase(memory.NewClock(), events)
	getter := in.NewGetConversationUseCase(projection)
	lister := in.NewListConversationEventsUseCase(events)
	byParticipant := in.NewGetConversationsByParticipantUseCase(projection)

	return httpapi.NewConversationHandler(starter, adder, replier, participants, getter, lister, byParticipant)
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
	assert.NoErr(t, err, "StartConversation")

	_, ok := got.(httpapi.StartConversation400JSONResponse)
	assert.True(t, ok, "StartConversation with no resourceUrl = %#v, want a StartConversation400JSONResponse", got)
}

func TestConversationHandler_AddThread_MissingAuthorIsRejected(t *testing.T) {
	handler := conversationHandler(t)

	got, err := handler.AddThread(context.Background(), httpapi.AddThreadRequestObject{
		ConversationId: "conversation-1",
		Body: &httpapi.AddThreadJSONRequestBody{
			ThreadTitle: strPtr("Delivery date"),
			Recipients:  &[]string{"user-4"},
			Message:     strPtr("When will this ship?"),
		},
	})
	assert.NoErr(t, err, "AddThread")

	_, ok := got.(httpapi.AddThread400JSONResponse)
	assert.True(t, ok, "AddThread with no author = %#v, want an AddThread400JSONResponse", got)
}

func TestConversationHandler_AddThread_UnknownConversationIs404(t *testing.T) {
	handler := conversationHandler(t)

	got, err := handler.AddThread(context.Background(), httpapi.AddThreadRequestObject{
		ConversationId: "does-not-exist",
		Body: &httpapi.AddThreadJSONRequestBody{
			ThreadTitle: strPtr("Delivery date"),
			Author:      strPtr("user-3"),
			Recipients:  &[]string{"user-4"},
			Message:     strPtr("When will this ship?"),
		},
	})
	assert.NoErr(t, err, "AddThread")

	_, ok := got.(httpapi.AddThread404JSONResponse)
	assert.True(t, ok, "AddThread(%q) = %#v, want an AddThread404JSONResponse", "does-not-exist", got)
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
	assert.NoErr(t, err, "ReplyToThread")

	_, ok := got.(httpapi.ReplyToThread400JSONResponse)
	assert.True(t, ok, "ReplyToThread with no author = %#v, want a ReplyToThread400JSONResponse", got)
}

func TestConversationHandler_AddThreadParticipant_MissingParticipantIDIsRejected(t *testing.T) {
	handler := conversationHandler(t)

	got, err := handler.AddThreadParticipant(context.Background(), httpapi.AddThreadParticipantRequestObject{
		ConversationId: "conversation-1",
		ThreadId:       "thread-1",
	})
	assert.NoErr(t, err, "AddThreadParticipant")

	_, ok := got.(httpapi.AddThreadParticipant400JSONResponse)
	assert.True(t, ok, "AddThreadParticipant with no participantId = %#v, want an AddThreadParticipant400JSONResponse", got)
}

func TestConversationHandler_RemoveThreadParticipant_MissingParticipantIDIsRejected(t *testing.T) {
	handler := conversationHandler(t)

	got, err := handler.RemoveThreadParticipant(context.Background(), httpapi.RemoveThreadParticipantRequestObject{
		ConversationId: "conversation-1",
		ThreadId:       "thread-1",
	})
	assert.NoErr(t, err, "RemoveThreadParticipant")

	_, ok := got.(httpapi.RemoveThreadParticipant400JSONResponse)
	assert.True(t, ok, "RemoveThreadParticipant with no participantId = %#v, want a RemoveThreadParticipant400JSONResponse", got)
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
	assert.NoErr(t, err, "ReplyToThread")

	_, ok := got.(httpapi.ReplyToThread404JSONResponse)
	assert.True(t, ok, "ReplyToThread(%q) = %#v, want a ReplyToThread404JSONResponse", "does-not-exist", got)
}

func TestConversationHandler_ListConversationEvents_UnknownIDIs404(t *testing.T) {
	handler := conversationHandler(t)

	got, err := handler.ListConversationEvents(context.Background(), httpapi.ListConversationEventsRequestObject{ConversationId: "does-not-exist"})
	assert.NoErr(t, err, "ListConversationEvents")

	_, ok := got.(httpapi.ListConversationEvents404JSONResponse)
	assert.True(t, ok, "ListConversationEvents(%q) = %#v, want a ListConversationEvents404JSONResponse", "does-not-exist", got)
}

func TestConversationHandler_GetConversation_UnknownIDIs404(t *testing.T) {
	handler := conversationHandler(t)

	got, err := handler.GetConversation(context.Background(), httpapi.GetConversationRequestObject{Id: "does-not-exist"})
	assert.NoErr(t, err, "GetConversation")

	_, ok := got.(httpapi.GetConversation404JSONResponse)
	assert.True(t, ok, "GetConversation(%q) = %#v, want a GetConversation404JSONResponse", "does-not-exist", got)
}
