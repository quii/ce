// Package postgrestest starts a real, ephemeral Postgres via
// testcontainers and applies every migration - shared by the postgres
// out-port contract tests and the container driver's topology
// (specifications/container), both of which need a real schema on a real
// Postgres before anything runs against it, see
// docs/adr/0026-sql-spec-first-with-sqlc.md.
package postgrestest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/quii/ce/internal/adapters/postgres"
	"github.com/quii/ce/internal/assert"
)

const (
	user     = "ce"
	password = "ce"
	dbName   = "ce"
)

// StartContainer returns a connection string rather than an already-open
// pool or *Store: its two callers need different things built from it -
// the contract tests need a raw pool for direct SQL (TRUNCATE between
// cases), while a test proving Postgres round-trips land in UTC needs the
// same postgres.NewPool production code path uses - so building either
// one here would just be a second, redundant construction for the other.
func StartContainer(t *testing.T) string {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "postgres:17-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     user,
			"POSTGRES_PASSWORD": password,
			"POSTGRES_DB":       dbName,
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").
			WithOccurrence(2).
			WithStartupTimeout(2 * time.Minute),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	assert.NoErr(t, err, "start postgres container")
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	host, err := container.Host(ctx)
	assert.NoErr(t, err, "get postgres container host")
	port, err := container.MappedPort(ctx, "5432")
	assert.NoErr(t, err, "get postgres mapped port")

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port.Port(), dbName)

	assert.NoErr(t, postgres.Migrate(ctx, connString), "migrate postgres container")

	return connString
}
