# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

`make` targets load `.env` if present. Copy `.env.example` first. Without make:
`docker compose up -d --wait database`, `go run ./cmd/migrate up`, `go run ./cmd/api`.

```sh
make up / down / clean-db     # Postgres 18 (service name: database)
make run                      # serve on :8080
make test                     # unit tests, no Docker, no build tag
make test-integration         # -tags integration, real Postgres via testcontainers
make test-one PKG=./internal/application RUN=TestDeposit
make lint                     # golangci-lint on both build configurations
make generate                 # sqlc + wire
make migrate-up / migrate-create NAME=x
```

The frontend in `web/` is a separate npm project with its own commands; see
**The `web/` frontend** below. No make target touches it.

Tests are split by build tag: anything needing Docker is behind `integration`.
`go vet -tags integration ./...` type-checks that tier without running it.

## Architecture

Dependencies point inward, and `depguard` in `.golangci.yml` enforces it rather
than leaving it to review. Adding an import that crosses a layer fails CI.

- `internal/domain/account/entity` -- the aggregate. All fields unexported; every
  invariant (no overdraft, no currency mixing, no closing a funded account) is a
  method here. Imports nothing from this module. `Money` is an immutable value
  object over integer minor units.
- `internal/domain/auth/entity` -- `User` and `Session`. One package because
  one bounded context, which is also what lets `rest/auth` keep to one domain
  import. Holds the password *policy* and session expiry, and no cryptography.
- `internal/application` -- use cases. Thin: parse to a value object, call one
  aggregate method, persist. Owns the transaction boundary via the `TxManager`
  port. Growing conditionals here means a rule belongs in the domain.
- `internal/infrastructure` -- adapters implementing the inner ports:
  `persistence` (pgx + sqlc), `system` (clock, ids), `security` (Argon2id,
  `crypto/rand` tokens).
- `internal/interfaces/rest` -- the router, the middleware stack, and the
  operational endpoints. Each resource is a subpackage (`rest/account`,
  `rest/auth`, `rest/post`) holding its handler, DTOs, and error mapping;
  `rest/httpx` holds what they share. `rest/auth` additionally exports
  `RequireSession`, the middleware every other resource is mounted behind.

**A REST package imports exactly one domain package.** That is the rule the
split exists to enforce, and `depguard` enforces half of it: `rest/httpx` may
import no domain at all. Every domain package is named `entity`, so a file that
needs two aggregates cannot name them both -- if a helper needs that, it is
transport plumbing and belongs in `httpx` phrased in transport terms
(`httpx.Page`, not `entity.Page`; `httpx.Status`, not an HTTP status per
aggregate). Adding a resource means adding a directory, not prefixing names.

Ports are declared by the consumer, not the provider: `entity.Repository`/
`entity.UserRepository`/`entity.SessionRepository` in the domain,
`application.TxManager`/`Clock`/`IDGenerator`/`PasswordHasher`/`TokenGenerator`
in the application layer, `account.Service`/`auth.Service`/`post.Service`/
`rest.Pinger` in the REST layer. `rest.Pinger` exists so the interfaces layer
never imports pgx.

A REST-layer port returns domain types only, never application types -- that is
why signing in returns `entity.Credential` rather than a struct owned by
`application`.

### Authentication

Session cookies, not JWTs. `/v1/auth/register` and `/v1/auth/login` are the only
routes under `/v1` outside `RequireSession`; `/healthz` and `/readyz` are
outside it too, because a probe has no session.

Invariants worth not breaking:

- **The plaintext session token is never stored.** `sessions.token_hash` holds a
  SHA-256 digest, and a CHECK constraint enforces the shape. Lookup is by
  digest, which is why the token hash is a plain unsalted SHA-256 and the
  *password* hash is Argon2id -- one needs to be a stable key, the other needs
  to be slow.
- **Login collapses every failure into `entity.ErrInvalidCredentials`**, and
  verifies against a decoy hash when the address is unknown so the timing
  matches. Do not add a branch that distinguishes them anywhere -- not in the
  service, not in `statusFor`.
- **`SameSite=Lax` is the CSRF defence**, so no `GET` may change state.
- **Changing a password calls `sessions.DeleteByUser` and re-issues** the
  caller's cookie in the same transaction.
- Registration always creates a `member`. Nothing grants `admin` over HTTP.

`config.SecureCookies()` forces `Secure` on in production regardless of
`SESSION_COOKIE_SECURE`; build cookies through `authrest.CookieConfig` (wired in
`internal/providers.go`) rather than reading the config field directly.

`internal/providers.go` holds the config-narrowing providers. They cannot live
in `wire.go`, which is behind the `wireinject` tag and never compiled into the
binary.

### Wire

`internal/wire.go` splits the graph: `coreSet` takes a `*pgxpool.Pool` as given,
`InitializeAPI` adds `postgres.New`, `InitializeAPIWithPool` does not. The second
injector is what lets integration tests run the real wiring on a container pool
without a second object graph. `wire_gen.go` is generated -- run `make wire`.

### Persistence

No ORM. SQL lives in `db/queries/`, `sqlc generate` produces
`internal/infrastructure/persistence/sqlcgen/` (generated, do not edit).

Writes use optimistic locking: `UpdateAccount` is `:execrows` guarded by
`version`, and the repository maps 0 rows to `entity.ErrConflict` (HTTP 409).
When adding a mutating query, keep that guard.

`TxManager.WithinTx` puts the pgx transaction on the context; `queries(ctx, pool)`
in `persistence/tx.go` picks it up, so repository methods work inside or outside
a transaction unchanged. It is reentrant: nested calls join the outer transaction.

Migrations carry `CHECK` constraints mirroring the domain invariants as defence
in depth. Change both together.

## Conventions

- Wrap errors with the call site: `fmt.Errorf("Type - Method - dep.Call: %w", err)`.
- Domain errors are sentinels matched with `errors.Is`. To add one: define it in
  the entity package and add a case to that resource's `statusFor`
  (`rest/account/errors.go`, `rest/post/errors.go`), the only place in its
  package that knows both the domain and HTTP vocabularies. A `statusFor`
  returns `false` for errors it does not own; `httpx.WriteError` maps those to
  an opaque 500, so a sentinel that is never added is a silent 500.
- Transport-level rejections use `httpx.NewRequestError`, not a domain sentinel:
  "invalid JSON" is not a statement about accounts.
- Never call `time.Now()` or `uuid.New()` outside `internal/infrastructure/system`.
  Use the injected `Clock` and `IDGenerator` so tests stay deterministic.
- Aggregate methods take `now time.Time` rather than reading the clock.
- Request DTOs use pointers for required numeric fields so a missing value is
  distinguishable from an explicit zero.
- `golangci-lint` is v2 (`version: "2"` schema, `default: none` plus an explicit
  enable list). v1 configs are not compatible. Suppressions need an explanation.

## Gotchas

- Migrations never run at startup; `cmd/migrate` is a separate binary.
- `/healthz` must not touch the database. Only `/readyz` does.
- `middleware.RealIP` is deliberately unused (spoofable, GHSA-3fxj-6jh8-hvhx).
- Bump the Go version in `go.mod`, `Dockerfile`, and `.github/workflows/ci.yml`
  together.

## The `web/` frontend

`web/` is a second, independent project: a React 19 + Vite + React Router v7 SPA
(shadcn/ui, Tailwind v4, TypeScript). It has its own `README.md`, `docs/`, and
`package.json`. It is **not** built, served, or embedded by the Go binary --
`vite build` emits a static `dist/`. Nothing in `go.mod` or the Makefile knows
it exists, so `make check` does not lint or typecheck it.

```sh
cd web && npm install
cp .env.example .env       # defaults are correct for local development
npm run dev                # :3000, proxying /v1 to the API on :8080
npm run build              # tsc --noEmit, then vite build
npm run typecheck / lint / format
```

Tooling is oxlint + oxfmt, not ESLint/Prettier. There is no test runner.

### Structure

`src/features/<name>/` is the unit of work, and each one layers
`api/types.ts` -> `api/service.ts` -> `api/queries.ts` (TanStack Query
`queryOptions` plus a `<name>Keys` key factory). Components import from
`queries.ts`/`service.ts`, never from the mocks. `pages/` holds one thin
component per route; `routes/index.tsx` is the declarative route tree;
`layouts/` holds the dashboard shell.

Being a port of a Next.js starter shapes the code: `components/link.tsx`,
`components/image.tsx`, `hooks/use-router.ts`, `hooks/use-pathname.ts`, and
`lib/not-found.ts` are shims that keep the Next-facing API (`Link href`,
`router.push`, `notFound()`) so copied components compile unchanged. Prefer the
shims over importing `react-router-dom` directly in components. The README's
conversion table is the reference for the rest.

- **Data is fake, except auth.** Every other feature reads in-memory faker mocks
  from `src/constants/mock-api*.ts`; `api/service.ts` is the single file to
  change per feature to hit a real backend, and `features/auth/api/service.ts`
  is the worked example of one that already does.
- **All API calls go through `src/lib/api.ts`.** Its `credentials: 'include'` is
  what sends the session cookie; a bare `fetch` is anonymous. The default base
  URL is empty so requests stay same-origin via the Vite proxy -- pointing
  `VITE_API_URL` at another origin breaks sign-in, because a `SameSite=Lax`
  cookie is withheld from cross-site mutations.
- **Forms** go through `useAppForm` from `lib/form.ts` (TanStack Form +
  createFormHook with every field pre-registered) and render inside
  `form.AppField`. Add a field component to `components/forms/fields/` and
  register it there rather than hand-rolling inputs. Zod v4 for schemas.
- **Themes** are `[data-theme]` CSS files in `src/styles/themes/`. Adding one
  touches four files -- see `docs/themes.md`.
- **Navigation and RBAC** are data: `src/config/nav-config.ts` drives both the
  sidebar and the Cmd+K bar, and each item's `access` object gates it. `access`
  has one field, `role`, checked against the session user. See
  `docs/nav-rbac.md`.
- **Session state is one React Query entry.** `sessionQueryOptions` wraps
  `GET /v1/auth/me`; `useSession()` in `hooks/use-session.ts` is the only way to
  read it. There is no auth provider component. `protected-route.tsx` and
  `guest-route.tsx` guard the two halves of the route tree, and both treat
  "pending" as distinct from "signed out" -- collapsing them bounces every user
  to sign-in on a hard refresh.
- **The route guards are UX, not a security boundary.** The Go API enforces
  access; anything the guard hides is still one `curl` away without it.
- **Deep links need a host rewrite** to `/index.html`, or `/dashboard/product/12`
  404s in production.

The oxlint React Compiler rules (`react/purity`, `react/set-state-in-effect`,
`react/exhaustive-effect-dependencies`, ...) are set to `warn` because dropping
the Next `'use client'` directives switched them on across the whole tree. New
code should not add to the pile.
