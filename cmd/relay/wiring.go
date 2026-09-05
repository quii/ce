package main

import (
	"context"

	"github.com/quii/ce/internal/adapters/postgres"
	"github.com/quii/ce/internal/ports/in"
	"github.com/quii/ce/internal/ports/out"
)

type OutPorts struct {
	Outbox     out.Outbox
	Projection out.Projection
}

func NewOutPorts(ctx context.Context, databaseURL string) (*OutPorts, error) {
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	store := postgres.NewStore(pool)
	return &OutPorts{
		Outbox:     store,
		Projection: store,
	}, nil
}

type Application struct {
	Drain in.Relay
}

func NewApplication(ports *OutPorts) *Application {
	return &Application{
		Drain: in.NewRelay(ports.Outbox, ports.Projection),
	}
}
