# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this repository is

Two things that have not been joined up yet:

- **The root** is the product definition. `README.md` is the source of truth for
  the domain (Person, Relationship, Interaction, RelationshipHealth, Reminder,
  ImportantEvent), the MVP checklist, and the 5-phase roadmap. `AGENTS.md` is a
  shorter agent-facing summary of it.
- **`app/`** is the implementation skeleton, added later and **not yet tracked by
  git** (`git status` shows it untracked; `git ls-files` lists only the two
  markdown files). It is a working Go + React stack. See `app/CLAUDE.md`, which
  is detailed and current, for every command and architectural rule.

Because of that split, the two halves disagree and neither is wrong:

- `AGENTS.md` still says "No source code yet". It predates `app/` and should not
  be trusted about the current state; trust `app/CLAUDE.md` instead.
- `app/` carries **none of the Relationship Graph domain**. Its Go module is
  `github.com/kethaka-creskit/go-ddd-service`, and its aggregates are `account`
  (a bank account, with real invariants chosen to exercise DDD), `post`, and
  `auth`. `account` and `post` are placeholders to be replaced by the README's
  entities; `auth` (users, sessions, cookies) is real infrastructure to keep.

So work on this repo is usually one of: designing the domain against `README.md`,
or growing `app/` toward it by adding a new aggregate the way `account` is built.

## Commands

All development happens inside `app/`. There is no build at the repository root.

```sh
cd app && make help          # Go API: up / migrate-up / run / test / lint / generate
cd app/web && npm run dev    # React SPA on :3000, proxying /v1 to the API on :8080
```

`app/CLAUDE.md` documents both in full, including how tests are split by the
`integration` build tag and which parts of the frontend are faker mocks.

## Adding a Relationship Graph aggregate

The stack's layering is enforced by `depguard` in `app/.golangci.yml`, so
following `account` end to end is the fastest correct path. One aggregate spans:

`db/migrations/` (with CHECK constraints mirroring the domain invariants) →
`db/queries/` → `make sqlc` → `internal/domain/<name>/entity/` (all fields
unexported, invariants as methods, `Repository` port declared here) →
`internal/infrastructure/persistence/` → `internal/application/` (thin use cases,
transaction boundary) → `internal/interfaces/rest/<name>/` (handler, DTOs,
`statusFor` error mapping) → mount in `internal/interfaces/rest/router.go` behind
`RequireSession` → `internal/providers.go` + `make wire`.

Two places where the README's domain will push against the skeleton, worth
deciding deliberately rather than discovering mid-implementation:

- A `Relationship` is an edge between two `Person` rows, and graph queries
  (mutual connections, connection paths) are recursive. `sqlc` handles recursive
  CTEs fine, but they belong in `db/queries/`, not assembled in Go.
- `RelationshipHealth` is a derived score. It is a domain concept (a value object
  over recency, frequency, importance), not a database column, so compute it in
  `internal/domain/`, the way `Money` is a value object rather than an int.
