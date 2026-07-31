package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

// GetConversationsByParticipantCommand is raw, not-yet-validated input -
// see docs/adr/0010-tiny-types.md.
type GetConversationsByParticipantCommand struct {
	ParticipantID string
}

type ConversationsByParticipantGetter interface {
	GetConversationsByParticipant(ctx context.Context, cmd GetConversationsByParticipantCommand) ([]domain.ConversationView, error)
}

type getConversationsByParticipantUseCase struct {
	projection out.Projection
}

func NewGetConversationsByParticipantUseCase(projection out.Projection) ConversationsByParticipantGetter {
	return &getConversationsByParticipantUseCase{projection: projection}
}

func (uc *getConversationsByParticipantUseCase) GetConversationsByParticipant(ctx context.Context, cmd GetConversationsByParticipantCommand) ([]domain.ConversationView, error) {
	return uc.projection.GetByParticipant(ctx, domain.ParticipantID(cmd.ParticipantID))
}
