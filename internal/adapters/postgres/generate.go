// Package postgres holds the Postgres-backed out-port adapters. The
// *.sql.go files and models.go are generated from migrations/*.sql (the
// schema) and queries/*.sql (named queries) - see
// docs/adr/0026-sql-spec-first-with-sqlc.md. Run `go generate ./...` after
// editing either.
//
//go:generate go tool sqlc generate -f sqlc.yaml
package postgres
