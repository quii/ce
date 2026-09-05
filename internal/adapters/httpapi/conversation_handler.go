package httpapi

import (
	"context"
	"errors"
	"fmt"
	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

type ConversationHandler struct {
	starter       in.ConversationStarter
	adder         in.ThreadAdder
	replier       in.ThreadReplier
	participants  in.ThreadParticipantManager
	getter        in.ConversationGetter
	lister        in.EventLister
	byParticipant in.ConversationsByParticipantGetter
}

func NewConversationHandler(starter in.ConversationStarter, adder in.ThreadAdder, replier in.ThreadReplier, participants in.ThreadParticipantManager, getter in.ConversationGetter, lister in.EventLister, byParticipant in.ConversationsByParticipantGetter) *ConversationHandler {
	return &ConversationHandler{starter: starter, adder: adder, replier: replier, participants: participants, getter: getter, lister: lister, byParticipant: byParticipant}
}

func (h *ConversationHandler) AddThreadParticipant(ctx context.Context, request AddThreadParticipantRequestObject) (AddThreadParticipantResponseObject, error) {
	result, err := h.participants.AddThreadParticipant(ctx, in.ManageThreadParticipantCommand{ConversationID: request.ConversationId, ThreadID: request.ThreadId, ParticipantID: request.ParticipantId})
	if err != nil {
		switch kind, message := classifyDomainError(err, domain.ErrThreadNotFound); kind {
		case validationErrorKind:
			return AddThreadParticipant400JSONResponse{Message: message}, nil
		case notFoundErrorKind:
			return AddThreadParticipant404JSONResponse{Message: message}, nil
		default:
			return nil, err
		}
	}
	if !result.Changed {
		return AddThreadParticipant204Response{}, nil
	}
	location := fmt.Sprintf("/conversations/%s?after=%d", result.ConversationID, result.Sequence)
	return AddThreadParticipant202Response{Headers: AddThreadParticipant202ResponseHeaders{Location: &location}}, nil
}

func (h *ConversationHandler) RemoveThreadParticipant(ctx context.Context, request RemoveThreadParticipantRequestObject) (RemoveThreadParticipantResponseObject, error) {
	result, err := h.participants.RemoveThreadParticipant(ctx, in.ManageThreadParticipantCommand{ConversationID: request.ConversationId, ThreadID: request.ThreadId, ParticipantID: request.ParticipantId})
	if err != nil {
		switch kind, message := classifyDomainError(err, domain.ErrThreadNotFound); kind {
		case validationErrorKind:
			return RemoveThreadParticipant400JSONResponse{Message: message}, nil
		case notFoundErrorKind:
			return RemoveThreadParticipant404JSONResponse{Message: message}, nil
		default:
			return nil, err
		}
	}
	if !result.Changed {
		return RemoveThreadParticipant204Response{}, nil
	}
	location := fmt.Sprintf("/conversations/%s?after=%d", result.ConversationID, result.Sequence)
	return RemoveThreadParticipant202Response{Headers: RemoveThreadParticipant202ResponseHeaders{Location: &location}}, nil
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

func (h *ConversationHandler) GetConversationsByParticipant(ctx context.Context, request GetConversationsByParticipantRequestObject) (GetConversationsByParticipantResponseObject, error) {
	views, err := h.byParticipant.GetConversationsByParticipant(ctx, in.GetConversationsByParticipantCommand{
		ParticipantID: request.Params.ParticipantId,
	})
	if err != nil {
		return nil, err
	}

	conversations := make([]Conversation, len(views))
	for i, v := range views {
		conversations[i] = toConversation(v)
	}

	return GetConversationsByParticipant200JSONResponse(conversations), nil
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
