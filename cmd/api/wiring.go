package main

import (
	"context"

	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/adapters/postgres"
	"github.com/quii/ce/internal/ports/in"
	"github.com/quii/ce/internal/ports/out"
)

type OutPorts struct {
	IDs        out.IDGenerator
	Clock      out.Clock
	Events     out.EventStore
	Projection out.Projection
}

func NewOutPorts(ctx context.Context, databaseURL string) (*OutPorts, error) {
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	store := postgres.NewStore(pool)
	return &OutPorts{
		IDs:        memory.NewIDGenerator(),
		Clock:      memory.NewClock(),
		Events:     store,
		Projection: store,
	}, nil
}

type Application struct {
	StartConversation             in.ConversationStarter
	AddThread                     in.ThreadAdder
	ReplyToThread                 in.ThreadReplier
	ManageThreadParticipants      in.ThreadParticipantManager
	GetConversation               in.ConversationGetter
	ListConversationEvents        in.EventLister
	GetConversationsByParticipant in.ConversationsByParticipantGetter
}

func NewApplication(ports *OutPorts) *Application {
	return &Application{
		StartConversation:             in.NewStartConversationUseCase(ports.IDs, ports.Clock, ports.Events),
		AddThread:                     in.NewAddThreadUseCase(ports.IDs, ports.Clock, ports.Events, ports.Projection),
		ReplyToThread:                 in.NewReplyToThreadUseCase(ports.IDs, ports.Clock, ports.Events, ports.Projection),
		ManageThreadParticipants:      in.NewManageThreadParticipantUseCase(ports.Clock, ports.Events),
		GetConversation:               in.NewGetConversationUseCase(ports.Projection),
		ListConversationEvents:        in.NewListConversationEventsUseCase(ports.Events),
		GetConversationsByParticipant: in.NewGetConversationsByParticipantUseCase(ports.Projection),
	}
}
