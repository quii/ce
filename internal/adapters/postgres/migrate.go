package postgres

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"

	_ "github.com/jackc/pgx/v5/stdlib" // registers the "pgx" database/sql driver goose needs
	"github.com/pressly/goose/v3"
	"github.com/pressly/goose/v3/lock"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate applies any pending migrations, guarded by a Postgres
// session-level advisory lock so concurrent replicas starting at the same
// time don't race each other - see
// docs/adr/0026-sql-spec-first-with-sqlc.md. cmd/api calls this before
// serving traffic; test setup (contract tests, the container driver) calls
// the same function against a testcontainers-provisioned Postgres.
func Migrate(ctx context.Context, connString string) error {
	db, err := sql.Open("pgx", connString)
	if err != nil {
		return fmt.Errorf("could not open a database/sql connection for migrations: %w", err)
	}
	defer func() { _ = db.Close() }()

	migrations, err := fs.Sub(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("could not load embedded migrations: %w", err)
	}

	locker, err := lock.NewPostgresSessionLocker()
	if err != nil {
		return fmt.Errorf("could not create migration advisory lock: %w", err)
	}

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations, goose.WithSessionLocker(locker))
	if err != nil {
		return fmt.Errorf("could not create migration provider: %w", err)
	}

	if _, err := provider.Up(ctx); err != nil {
		return fmt.Errorf("could not apply migrations: %w", err)
	}

	return nil
}
