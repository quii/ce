package main

import (
	"context"

	"github.com/quii/ce/internal/adapters/postgres"
	"github.com/quii/ce/internal/ports/in"
	"github.com/quii/ce/internal/ports/out"
)

// docs/adr/0025-composition-root.md: the only place a concrete out-adapter gets constructed.
type OutPorts interface {
	out.Outbox
	out.Projection
}

type outPorts struct {
	out.Outbox
	out.Projection
}

// NewOutPorts wires cmd/relay's production out-ports to the same Postgres
// database cmd/api writes to - see docs/adr/0026-sql-spec-first-with-sqlc.md.
// Migrations are applied by cmd/api's startup, not here (per that ADR);
// the relay just reads/writes the schema api already migrated.
func NewOutPorts(ctx context.Context, databaseURL string) (OutPorts, error) {
	pool, err := postgres.NewPool(ctx, databaseURL)
	if err != nil {
		return nil, err
	}

	store := postgres.NewStore(pool)
	return &outPorts{
		Outbox:     store,
		Projection: store,
	}, nil
}

type Application struct {
	Drain in.Relay
}

func NewApplication(ports OutPorts) *Application {
	return &Application{
		Drain: in.NewRelay(ports, ports),
	}
}
