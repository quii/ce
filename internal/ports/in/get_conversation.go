package in

import (
	"context"

	"github.com/quii/ce/internal/domain"
	"github.com/quii/ce/internal/ports/out"
)

type GetConversationCommand struct {
	ConversationID string
	After          *int64
}

type ConversationGetter interface {
	GetConversation(ctx context.Context, cmd GetConversationCommand) (domain.ConversationView, error)
}

type getConversationUseCase struct {
	projection out.Projection
}

func NewGetConversationUseCase(projection out.Projection) ConversationGetter {
	return &getConversationUseCase{projection: projection}
}

func (uc *getConversationUseCase) GetConversation(ctx context.Context, cmd GetConversationCommand) (domain.ConversationView, error) {
	if cmd.After != nil {
		checkpoint, err := uc.projection.Checkpoint(ctx)
		if err != nil {
			return domain.ConversationView{}, err
		}
		if int64(checkpoint) < *cmd.After {
			return domain.ConversationView{}, domain.ErrProjectionNotCaughtUp
		}
	}

	return uc.projection.Get(ctx, domain.ConversationID(cmd.ConversationID))
}
