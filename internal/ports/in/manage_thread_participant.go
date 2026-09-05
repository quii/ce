package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

type ManageThreadParticipantCommand struct {
	ConversationID string
	ThreadID       string
	ParticipantID  string
}

type ManageThreadParticipantResult struct {
	ConversationID domain.ConversationID
	Sequence       domain.Sequence
	Changed        bool
}

type ThreadParticipantManager interface {
	AddThreadParticipant(context.Context, ManageThreadParticipantCommand) (ManageThreadParticipantResult, error)
	RemoveThreadParticipant(context.Context, ManageThreadParticipantCommand) (ManageThreadParticipantResult, error)
}

type manageThreadParticipantUseCase struct {
	clock  out.Clock
	events out.EventStore
}

func NewManageThreadParticipantUseCase(clock out.Clock, events out.EventStore) ThreadParticipantManager {
	return &manageThreadParticipantUseCase{clock: clock, events: events}
}

func (uc *manageThreadParticipantUseCase) AddThreadParticipant(ctx context.Context, cmd ManageThreadParticipantCommand) (ManageThreadParticipantResult, error) {
	return uc.manage(ctx, cmd, domain.Conversation.AddParticipant)
}

func (uc *manageThreadParticipantUseCase) RemoveThreadParticipant(ctx context.Context, cmd ManageThreadParticipantCommand) (ManageThreadParticipantResult, error) {
	return uc.manage(ctx, cmd, domain.Conversation.RemoveParticipant)
}

func (uc *manageThreadParticipantUseCase) manage(
	ctx context.Context,
	cmd ManageThreadParticipantCommand,
	action func(domain.Conversation, domain.ManageThreadParticipantParams) (domain.Event, bool, error),
) (ManageThreadParticipantResult, error) {
	conversationID := domain.ConversationID(cmd.ConversationID)
	params := domain.ManageThreadParticipantParams{
		ConversationID: conversationID,
		ThreadID:       domain.ThreadID(cmd.ThreadID),
		ParticipantID:  domain.ParticipantID(cmd.ParticipantID),
	}
	if err := domain.ValidateManageThreadParticipant(params); err != nil {
		return ManageThreadParticipantResult{}, err
	}
	params.OccurredAt = uc.clock.Now()

	records, err := uc.events.ListByConversation(ctx, conversationID)
	if err != nil {
		return ManageThreadParticipantResult{}, err
	}
	conversation, err := domain.RehydrateConversation(records)
	if err != nil {
		return ManageThreadParticipantResult{}, err
	}

	event, changed, err := action(conversation, params)
	if err != nil {
		return ManageThreadParticipantResult{}, err
	}
	if !changed {
		return ManageThreadParticipantResult{ConversationID: conversationID}, nil
	}

	seq, err := uc.events.Append(ctx, event)
	if err != nil {
		return ManageThreadParticipantResult{}, err
	}
	return ManageThreadParticipantResult{ConversationID: conversationID, Sequence: seq, Changed: true}, nil
}
