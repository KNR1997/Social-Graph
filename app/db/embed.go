// Package db embeds the migration files so the migrate command is a single
// self-contained binary: no directory to ship alongside it, no path to get
// wrong in a container. go:embed cannot reach into a parent directory, which is
// why this lives here rather than in cmd/migrate.
package db

import "embed"

// Migrations holds the goose migration files.
//
//go:embed migrations/*.sql
var Migrations embed.FS

// MigrationsDir is the path to the migrations inside Migrations.
const MigrationsDir = "migrations"

// SourceDir is the on-disk path to the migrations, relative to the module root.
// Migrations embed.FS is read-only, so `migrate create` writes here instead;
// it therefore only works when run from the repository (`go run ./cmd/migrate`),
// which is the only place new migrations are ever authored.
const SourceDir = "db/" + MigrationsDir
