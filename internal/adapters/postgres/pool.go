package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// NewPool is the one place a *pgxpool.Pool gets constructed for this
// module - cmd/api, cmd/relay and test setup (internal/adapters/postgres's
// own tests) all call this rather than pgxpool.New directly, so the UTC
// configuration below can't quietly regress in one caller and not
// another.
//
// pgx's timestamptz codec scans via time.Unix, which defaults to
// time.Local unless a connection's type map is told otherwise - without
// this, every timestamptz read back from Postgres would carry a local
// location instead of UTC, violating docs/adr/0015-utc-always.md.
func NewPool(ctx context.Context, connString string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("could not parse postgres connection string: %w", err)
	}

	cfg.AfterConnect = func(_ context.Context, conn *pgx.Conn) error {
		conn.TypeMap().RegisterType(&pgtype.Type{
			Name:  "timestamptz",
			OID:   pgtype.TimestamptzOID,
			Codec: &pgtype.TimestamptzCodec{ScanLocation: time.UTC},
		})
		return nil
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("could not connect to postgres: %w", err)
	}

	return pool, nil
}
