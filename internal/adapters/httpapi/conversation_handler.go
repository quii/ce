package httpapi

import (
	"context"
	"errors"
	"fmt"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/in"
)

type ConversationHandler struct {
	starter in.ConversationStarter
	replier in.ThreadReplier
	getter  in.ConversationGetter
}

func NewConversationHandler(starter in.ConversationStarter, replier in.ThreadReplier, getter in.ConversationGetter) *ConversationHandler {
	return &ConversationHandler{starter: starter, replier: replier, getter: getter}
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
		var validationErr domain.ValidationError
		if errors.As(err, &validationErr) {
			return StartConversation400JSONResponse{Message: validationErr.Error()}, nil
		}
		return nil, err
	}

	location := fmt.Sprintf("/conversations/%s?after=%d", result.ConversationID, result.Sequence)
	return StartConversation202Response{
		Headers: StartConversation202ResponseHeaders{Location: &location},
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
		var validationErr domain.ValidationError
		if errors.As(err, &validationErr) {
			return ReplyToThread400JSONResponse{Message: validationErr.Error()}, nil
		}
		if errors.Is(err, domain.ErrConversationNotFound) || errors.Is(err, domain.ErrThreadNotFound) {
			return ReplyToThread404JSONResponse{Message: err.Error()}, nil
		}
		if errors.Is(err, domain.ErrReplyForbidden) {
			return ReplyToThread403JSONResponse{Message: err.Error()}, nil
		}
		return nil, err
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

func toConversation(view domain.ConversationView) Conversation {
	participants := make([]string, len(view.Thread.Participants))
	for i, p := range view.Thread.Participants {
		participants[i] = string(p)
	}

	messages := make([]Message, len(view.Thread.Messages))
	for i, m := range view.Thread.Messages {
		messages[i] = Message{
			Author:   string(m.Author),
			Text:     string(m.Text),
			PostedAt: m.PostedAt,
		}
	}

	return Conversation{
		Id:          string(view.ID),
		ResourceUrl: string(view.ResourceURL),
		Thread: Thread{
			Id:           string(view.Thread.ID),
			Title:        string(view.Thread.Title),
			Participants: participants,
			Messages:     messages,
		},
	}
}
