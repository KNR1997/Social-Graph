//go:build integration

// Package containers starts throwaway infrastructure for integration tests.
//
// It is built only under the "integration" tag so that neither testcontainers
// nor the Docker client ends up in the dependency graph of a normal go test.
package containers

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib" // database/sql driver for goose
	"github.com/pressly/goose/v3"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/kethaka-creskit/go-ddd-service/db"
)

// postgresImage is pinned so a test run is reproducible and matches production.
const (
	postgresImage = "postgres:18-alpine"

	// containerStartTimeout allows for a cold image pull on the first run.
	containerStartTimeout = 60 * time.Second
	migrateTimeout        = 60 * time.Second

	// readyLogOccurrences: Postgres logs the ready line once during initdb and
	// again when it starts for real, so waiting for the second one avoids
	// connecting to the initdb instance.
	readyLogOccurrences = 2
)

// StartPostgres boots Postgres, applies every migration, and returns the DSN.
//
// Reuse is on: the container survives the test process so the next run attaches
// to it instead of paying the startup cost again, which is what makes this
// usable in a red-green loop rather than only in CI. Each caller gets a fresh
// schema regardless, via truncation in Truncate.
func StartPostgres(t *testing.T) string {
	t.Helper()

	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, postgresImage,
		tcpostgres.WithDatabase("app"),
		tcpostgres.WithUsername("app"),
		tcpostgres.WithPassword("secret"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(readyLogOccurrences).
				WithStartupTimeout(containerStartTimeout),
		),
		testcontainers.CustomizeRequest(testcontainers.GenericContainerRequest{
			ContainerRequest: testcontainers.ContainerRequest{Name: "go-ddd-service-test-postgres"},
			Reuse:            true,
		}),
	)
	if err != nil {
		t.Fatalf("start postgres container: %v", err)
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("container connection string: %v", err)
	}

	migrate(t, dsn)

	return dsn
}

// migrate applies the embedded goose migrations, exactly as the migrate command
// does in production. Tests must never carry their own copy of the schema: a
// second definition is a second thing to drift.
func migrate(t *testing.T, dsn string) {
	t.Helper()

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open migration connection: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	goose.SetBaseFS(db.Migrations)
	goose.SetTableName("schema_migrations")
	goose.SetLogger(goose.NopLogger())

	if err := goose.SetDialect("postgres"); err != nil {
		t.Fatalf("goose dialect: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), migrateTimeout)
	defer cancel()

	if err := goose.UpContext(ctx, sqlDB, db.MigrationsDir); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
}

// Truncate empties the tables so each test starts from a known state. This is
// far faster than recreating the container or re-running migrations per test.
func Truncate(t *testing.T, dsn string, tables ...string) {
	t.Helper()

	ctx := context.Background()

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open truncate connection: %v", err)
	}
	defer func() { _ = sqlDB.Close() }()

	for _, table := range tables {
		// #nosec G202 -- table names are test-supplied constants, never user input.
		if _, err := sqlDB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			t.Fatalf("truncate %s: %v", table, err)
		}
	}
}
