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
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}
	t.Cleanup(func() {
		_ = container.Terminate(ctx)
	})

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get postgres container host: %v", err)
	}
	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("failed to get postgres mapped port: %v", err)
	}

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, password, host, port.Port(), dbName)

	if err := postgres.Migrate(ctx, connString); err != nil {
		t.Fatalf("failed to migrate postgres container: %v", err)
	}

	return connString
}
