package httpapi

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

type ConversationHandler struct {
	starter in.ConversationStarter
	adder   in.ThreadAdder
	replier in.ThreadReplier
	getter  in.ConversationGetter
	lister  in.EventLister
}

func NewConversationHandler(starter in.ConversationStarter, adder in.ThreadAdder, replier in.ThreadReplier, getter in.ConversationGetter, lister in.EventLister) *ConversationHandler {
	return &ConversationHandler{starter: starter, adder: adder, replier: replier, getter: getter, lister: lister}
}

func (h *ConversationHandler) StartConversation(ctx context.Context, request StartConversationRequestObject) (StartConversationResponseObject, error) {
	cmd := in.StartConversationCommand{}
	if request.Body != nil {
		cmd.ResourceURL = request.Body.ResourceUrl
		cmd.ThreadTitle = request.Body.ThreadTitle
		cmd.Author = request.Body.Author
		cmd.Recipients = request.Body.Recipients
		cmd.Message = request.Body.Message
	}

	result, err := h.starter.StartConversation(ctx, cmd)
	if err != nil {
		if kind, message := classifyDomainError(err, nil); kind == validationErrorKind {
			return StartConversation400JSONResponse{Message: message}, nil
		}
		return nil, err
	}

	location := fmt.Sprintf("/conversations/%s?after=%d", result.ConversationID, result.Sequence)
	return StartConversation202Response{
		Headers: StartConversation202ResponseHeaders{Location: &location},
	}, nil
}

func (h *ConversationHandler) AddThread(ctx context.Context, request AddThreadRequestObject) (AddThreadResponseObject, error) {
	cmd := in.AddThreadCommand{ConversationID: request.ConversationId}
	if request.Body != nil {
		cmd.ThreadTitle = request.Body.ThreadTitle
		cmd.Author = request.Body.Author
		cmd.Recipients = request.Body.Recipients
		cmd.Message = request.Body.Message
	}

	result, err := h.adder.AddThread(ctx, cmd)
	if err != nil {
		switch kind, message := classifyDomainError(err, nil); kind {
		case validationErrorKind:
			return AddThread400JSONResponse{Message: message}, nil
		case notFoundErrorKind:
			return AddThread404JSONResponse{Message: message}, nil
		default:
			return nil, err
		}
	}

	location := fmt.Sprintf("/conversations/%s?after=%d", result.ConversationID, result.Sequence)
	return AddThread202Response{
		Headers: AddThread202ResponseHeaders{Location: &location},
	}, nil
}

func (h *ConversationHandler) ReplyToThread(ctx context.Context, request ReplyToThreadRequestObject) (ReplyToThreadResponseObject, error) {
	cmd := in.ReplyToThreadCommand{
		ConversationID: request.ConversationId,
		ThreadID:       request.ThreadId,
	}
	if request.Body != nil {
		cmd.Author = request.Body.Author
		cmd.Message = request.Body.Text
	}

	result, err := h.replier.ReplyToThread(ctx, cmd)
	if err != nil {
		switch kind, message := classifyDomainError(err, domain.ErrThreadNotFound); kind {
		case validationErrorKind:
			return ReplyToThread400JSONResponse{Message: message}, nil
		case notFoundErrorKind:
			return ReplyToThread404JSONResponse{Message: message}, nil
		case forbiddenErrorKind:
			return ReplyToThread403JSONResponse{Message: message}, nil
		default:
			return nil, err
		}
	}

	location := fmt.Sprintf("/conversations/%s?after=%d", result.ConversationID, result.Sequence)
	return ReplyToThread202Response{
		Headers: ReplyToThread202ResponseHeaders{Location: &location},
	}, nil
}

func (h *ConversationHandler) GetConversation(ctx context.Context, request GetConversationRequestObject) (GetConversationResponseObject, error) {
	view, err := h.getter.GetConversation(ctx, in.GetConversationCommand{
		ConversationID: request.Id,
		After:          request.Params.After,
	})
	if err != nil {
		if errors.Is(err, domain.ErrProjectionNotCaughtUp) {
			return GetConversation202Response{}, nil
		}
		if errors.Is(err, domain.ErrConversationNotFound) {
			return GetConversation404JSONResponse{Message: err.Error()}, nil
		}
		return nil, err
	}

	return GetConversation200JSONResponse(toConversation(view)), nil
}

func (h *ConversationHandler) ListConversationEvents(ctx context.Context, request ListConversationEventsRequestObject) (ListConversationEventsResponseObject, error) {
	records, err := h.lister.ListConversationEvents(ctx, in.ListConversationEventsCommand{ConversationID: request.ConversationId})
	if err != nil {
		if errors.Is(err, domain.ErrConversationNotFound) {
			return ListConversationEvents404JSONResponse{Message: err.Error()}, nil
		}
		return nil, err
	}

	events := make([]Event, len(records))
	for i, record := range records {
		events[i] = toEvent(record)
	}

	return ListConversationEvents200JSONResponse(events), nil
}

func toEvent(record domain.EventRecord) Event {
	event := Event{
		Sequence:   int64(record.Sequence),
		Type:       record.Event.TypeName(),
		OccurredAt: eventOccurredAt(record.Event),
	}

	switch e := record.Event.(type) {
	case domain.ConversationCreated:
		creator := string(e.Creator)
		resourceURL := string(e.ResourceURL)
		event.Creator = &creator
		event.ResourceUrl = &resourceURL
	case domain.ThreadStarted:
		threadID := string(e.ThreadID)
		threadTitle := string(e.ThreadTitle)
		author := string(e.Author)
		recipients := make([]string, len(e.Recipients))
		for i, r := range e.Recipients {
			recipients[i] = string(r)
		}
		event.ThreadId = &threadID
		event.ThreadTitle = &threadTitle
		event.Author = &author
		event.Recipients = &recipients
	case domain.MessagePosted:
		threadID := string(e.ThreadID)
		messageID := string(e.MessageID)
		author := string(e.Author)
		messageText := string(e.MessageText)
		event.ThreadId = &threadID
		event.MessageId = &messageID
		event.Author = &author
		event.MessageText = &messageText
	}

	return event
}

func eventOccurredAt(event domain.Event) time.Time {
	switch e := event.(type) {
	case domain.ConversationCreated:
		return e.OccurredAt
	case domain.ThreadStarted:
		return e.OccurredAt
	case domain.MessagePosted:
		return e.OccurredAt
	default:
		return time.Time{}
	}
}

func toConversation(view domain.ConversationView) Conversation {
	threads := make([]Thread, len(view.Threads))
	for i, t := range view.Threads {
		threads[i] = toThread(t)
	}

	return Conversation{
		Id:          string(view.ID),
		ResourceUrl: string(view.ResourceURL),
		Threads:     threads,
	}
}

func toThread(thread domain.ThreadView) Thread {
	participants := make([]string, len(thread.Participants))
	for i, p := range thread.Participants {
		participants[i] = string(p)
	}

	messages := make([]Message, len(thread.Messages))
	for i, m := range thread.Messages {
		messages[i] = Message{
			Author:   string(m.Author),
			Text:     string(m.Text),
			PostedAt: m.PostedAt,
		}
	}

	return Thread{
		Id:           string(thread.ID),
		Title:        string(thread.Title),
		Participants: participants,
		Messages:     messages,
	}
}
