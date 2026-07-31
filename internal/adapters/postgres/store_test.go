//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/quii/ce/internal/adapters/contracttest"
	"github.com/quii/ce/internal/adapters/postgres"
	"github.com/quii/ce/internal/adapters/postgres/postgrestest"
	"github.com/quii/ce/internal/assert"
	"github.com/quii/ce/internal/ports/out"
)

// TestStore_Contract runs the shared out.EventStore/out.Outbox/out.Projection
// contract suites against the real Postgres-backed adapter - see
// docs/adr/0009-contract-tests.md and docs/adr/0026-sql-spec-first-with-sqlc.md.
// One container is shared across all three suites; each suite truncates the
// schema before every subtest so it starts from the same clean slate the
// in-memory fake's constructor gives it.
func TestStore_Contract(t *testing.T) {
	connString := postgrestest.StartContainer(t)

	pool, err := postgres.NewPool(context.Background(), connString)
	assert.NoErr(t, err, "connect to postgres")
	t.Cleanup(pool.Close)

	t.Run("EventStore", func(t *testing.T) {
		contracttest.EventStore(t, func() out.EventStore {
			truncate(t, pool)
			return postgres.NewStore(pool)
		})
	})

	t.Run("Outbox", func(t *testing.T) {
		contracttest.Outbox(t, func() out.Outbox {
			truncate(t, pool)
			return postgres.NewStore(pool)
		})
	})

	t.Run("EventStoreOutbox", func(t *testing.T) {
		contracttest.EventStoreEnqueuesViaAppend(t, func() contracttest.EventStoreOutbox {
			truncate(t, pool)
			return postgres.NewStore(pool)
		})
	})

	t.Run("Projection", func(t *testing.T) {
		contracttest.Projection(t, func() out.Projection {
			truncate(t, pool)
			return postgres.NewStore(pool)
		})
	})

	t.Run("ProjectionByParticipant", func(t *testing.T) {
		contracttest.ProjectionByParticipant(t, func() out.Projection {
			truncate(t, pool)
			return postgres.NewStore(pool)
		})
	})
}

func truncate(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()

	_, err := pool.Exec(context.Background(),
		`TRUNCATE conversation_events, conversation_outbox, conversation_projection, conversation_projection_threads, conversation_projection_messages RESTART IDENTITY;
		 UPDATE projection_checkpoint SET sequence = 0`,
	)
	assert.NoErr(t, err, "truncate postgres tables between contract test cases")
}
