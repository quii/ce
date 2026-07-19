package main

import (
	"context"

	"github.com/quii/ce/internal/adapters/memory"
	"github.com/quii/ce/internal/adapters/postgres"
	"github.com/quii/ce/internal/ports/in"
	"github.com/quii/ce/internal/ports/out"
)

// docs/adr/0025-composition-root.md: the only place a concrete out-adapter gets constructed.
//
// out.Outbox isn't part of this bundle: nothing cmd/api wires up needs
// it - Events.Append alone durably writes the outbox row too (see
// in.StartConversationDependencies), and draining the outbox is
// cmd/relay's job, not the api role's. postgres.Store still implements
// out.Outbox (cmd/relay and the contract tests need that), this
// composition root just never reaches for it.
type OutPorts interface {
	out.GreetingFinder
	out.IDGenerator
	out.Clock
	out.EventStore
	out.Projection
}

type outPorts struct {
	out.GreetingFinder
	out.IDGenerator
	out.Clock
	out.EventStore
	out.Projection
}

// NewOutPorts wires cmd/api's production out-ports: the event store and
// projection are Postgres-backed (docs/adr/0026-sql-spec-first-with-sqlc.md);
// the ID generator and clock have no persistence concern of their own, so
// the same in-memory adapters that back the in-process driver/tests are
// what production uses too.
func NewOutPorts(ctx context.Context, databaseURL string) (OutPorts, error) {
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	store := postgres.NewStore(pool)
	return &outPorts{
		GreetingFinder: memory.NewGreetingFinder(),
		IDGenerator:    memory.NewIDGenerator(),
		Clock:          memory.NewClock(),
		EventStore:     store,
		Projection:     store,
	}, nil
}

type Application struct {
	GetGreeting       in.Greeter
	StartConversation in.ConversationStarter
	GetConversation   in.ConversationGetter
}

func NewApplication(ports OutPorts) *Application {
	return &Application{
		GetGreeting: in.NewGetGreetingUseCase(ports),
		StartConversation: in.NewStartConversationUseCase(in.StartConversationDependencies{
			IDs:    ports,
			Clock:  ports,
			Events: ports,
		}),
		GetConversation: in.NewGetConversationUseCase(ports),
	}
}
