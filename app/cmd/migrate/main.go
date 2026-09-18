// Command migrate applies goose migrations.
//
// Migrations are a deliberate, separate step: never run at service startup,
// where concurrent replicas race each other and a schema change becomes
// impossible to roll back independently of the deploy.
//
// Usage:
//
//	go run ./cmd/migrate up
//	go run ./cmd/migrate down
//	go run ./cmd/migrate status
//	go run ./cmd/migrate create add_something sql
package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"time"

	// database/sql driver used only by goose; the service itself talks pgx.
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"

	"github.com/kethaka-creskit/go-ddd-service/config"
	"github.com/kethaka-creskit/go-ddd-service/db"
)

// migrationTimeout bounds a single migrate invocation.
const migrationTimeout = 5 * time.Minute

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "migrate: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	args := os.Args[1:]
	if len(args) == 0 {
		return errors.New("usage: migrate <up|down|status|version|create> [args]")
	}

	// `create` only writes a file: no database, and therefore no config. It also
	// cannot use the embedded FS, which is read-only, so it targets the source
	// tree directly.
	if args[0] == "create" {
		return create(args[1:])
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// goose needs a database/sql handle, so open one alongside the pgx pool the
	// service uses. It lives only for the duration of this command.
	sqlDB, err := sql.Open("pgx", cfg.DB.URL)
	if err != nil {
		return fmt.Errorf("sql.Open: %w", err)
	}
	defer func() { _ = sqlDB.Close() }()

	goose.SetBaseFS(db.Migrations)
	goose.SetTableName("schema_migrations")

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("goose.SetDialect: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), migrationTimeout)
	defer cancel()

	if err := goose.RunContext(ctx, args[0], sqlDB, db.MigrationsDir, args[1:]...); err != nil {
		return fmt.Errorf("goose %s: %w", args[0], err)
	}

	return nil
}

// create writes a new blank migration into the source tree.
func create(args []string) error {
	if len(args) == 0 {
		return errors.New("usage: migrate create <name> [sql|go]")
	}

	migrationType := "sql"
	if len(args) > 1 {
		migrationType = args[1]
	}

	if err := goose.Create(nil, db.SourceDir, args[0], migrationType); err != nil {
		return fmt.Errorf("goose.Create: %w", err)
	}

	return nil
}
