# go-ddd-service

A Go backend scaffold laid out for Domain-Driven Design: chi for HTTP, pgx and
sqlc for data access with no ORM, goose for migrations, Postgres 18, and Wire
for compile-time dependency injection.

The example domain is a bank account, chosen because it has real invariants
(no overdraft, no currency mixing, no closing a funded account) rather than
being a bag of setters. Replace it with your own aggregate; keep the structure.

## Quick start

```sh
cp .env.example .env
make up            # Postgres 18, waits until it answers queries
make migrate-up    # apply migrations
make run           # serve on :8080
```

Then. Every route under `/v1` except `/v1/auth/register` and `/v1/auth/login`
needs a session, so the first call registers one and keeps the cookie (`-c`
writes the jar, `-b` sends it):

```sh
curl -s localhost:8080/healthz

curl -s -c jar -XPOST localhost:8080/v1/auth/register \
  -d '{"email":"ada@example.com","name":"Ada Lovelace","password":"correct-horse-battery"}'

curl -s -b jar localhost:8080/v1/auth/me

ACC=$(curl -s -b jar -XPOST localhost:8080/v1/accounts \
  -d '{"owner":"Ada Lovelace","currency":"USD"}' | jq -r .id)

curl -s -b jar -XPOST localhost:8080/v1/accounts/$ACC/deposit \
  -d '{"amount_minor":10000,"currency":"USD"}'

curl -s -b jar -XPOST localhost:8080/v1/accounts/$ACC/withdraw \
  -d '{"amount_minor":99999,"currency":"USD"}'   # 422 insufficient_funds

curl -s localhost:8080/v1/accounts                # 401 unauthenticated
```

`make` is optional. The equivalents are `docker compose up -d --wait database`,
`go run ./cmd/migrate up`, and `go run ./cmd/api`.

## Layout

```
cmd/api             process entry point: config, wiring, signals, shutdown
cmd/migrate         goose runner, a separate binary on purpose
config              the only place that reads the environment
db/migrations       goose migrations, embedded into both binaries
db/queries          hand-written SQL, the input to sqlc
internal/domain     aggregates, value objects, and the ports they need
internal/application use cases: orchestration and the transaction boundary
internal/infrastructure adapters: Postgres repositories, clock, ids, password hashing
internal/interfaces  inbound adapters: chi router, handlers, DTOs
internal/test        integration-test infrastructure (testcontainers)
pkg                  reusable plumbing: httpserver, logger, postgres
```

### The dependency rule

Dependencies point inward. `internal/domain` imports nothing from this module.
When the domain needs something from the outside, it **declares an interface**
and infrastructure implements it:

| Port (declared in) | Implementation |
| --- | --- |
| `entity.Repository` (domain) | `persistence.AccountRepository` |
| `entity.UserRepository`, `entity.SessionRepository` (domain) | `persistence.UserRepository`, `persistence.SessionRepository` |
| `application.TxManager` | `persistence.TxManager` |
| `application.Clock`, `application.IDGenerator` | `system.Clock`, `system.IDGenerator` |
| `application.PasswordHasher`, `application.TokenGenerator` | `security.Hasher`, `security.TokenGenerator` |
| `rest.AccountService`, `rest.Pinger` | `application.AccountService`, `*pgxpool.Pool` |

Note the last row: `rest` declares the interfaces it consumes, rather than the
application package exporting them. Consumer-side interfaces stay narrow and
make the handler testable with a stub.

This is not a convention you have to remember. `depguard` in `.golangci.yml`
enforces it, so a domain file that imports pgx fails CI:

```
import 'github.com/jackc/pgx/v5' is not allowed from list 'domain':
  no database driver in the domain (depguard)
```

### Where the rules live

Everything that can be wrong about an account is in
`internal/domain/account/entity`. `Account` has only unexported fields, so an
invalid instance cannot be constructed or observed: the balance cannot go
negative, currencies cannot mix, a frozen account rejects movement, a funded
account will not close. `Money` is an immutable value object holding integer
minor units, because floating-point money is a bug waiting for a rounding
error.

The application layer stays thin by design: parse input into a value object,
call one method on the aggregate, persist. If a use case starts growing
conditionals, the rule it is expressing belongs in the domain.

## Data access

No ORM. Write SQL in `db/queries/`, run `make sqlc`, and get typed Go:

```sql
-- name: UpdateAccount :execrows
UPDATE accounts
SET balance_minor = $2, status = $3, updated_at = $4, version = version + 1
WHERE id = $1 AND version = $5;
```

`:execrows` returns the affected row count, which the repository turns into
`entity.ErrConflict` when it is zero. That is the optimistic-locking scheme:
every write is guarded by the version it was read at, so a lost update becomes
a `409` instead of silently discarding someone else's deposit.

Migrations also carry `CHECK` constraints that mirror the domain invariants.
The domain is the source of truth; the constraints are defence in depth for
anything that reaches the table without going through the application, such as
a backfill or a psql session.

### Transactions

`application.TxManager` is a port. Its pgx implementation puts the transaction
on the context, and the repository picks it up from there, so no domain
signature mentions transactions. Nested calls join the outer transaction rather
than opening a second one. A use case therefore reads and writes atomically:

```go
err := s.tx.WithinTx(ctx, func(ctx context.Context) error {
    acc, err := s.accounts.ByID(ctx, id)   // same transaction
    if err != nil { return err }
    if err := acc.Withdraw(amount, s.clock.Now()); err != nil { return err }
    return s.accounts.Update(ctx, acc)     // same transaction
})
```

## Testing

Two tiers, split by build tag so the fast tier needs nothing installed:

```sh
make test              # unit tests, no Docker, no tag
make test-integration  # -tags integration, real Postgres via testcontainers
make test-all
make test-one PKG=./internal/application RUN=TestDeposit
```

The fast tier covers the aggregate directly (pure functions, no doubles needed),
the use cases against hand-written fakes, and the HTTP layer against a stub
service. The integration tier runs the repository, the transaction manager, and
the whole API against a real Postgres 18 container.

Every test is independently runnable: no shared package-level state initialised
by whichever test happens to run first, and `-run` on any single test works.

The container is started with `Reuse: true` and a fixed name, so it survives
between runs and the second run attaches instead of booting. Tables are
truncated per test. Migrations in tests come from the same embedded files the
migrate binary uses, so the test schema cannot drift from production.

## Code generation

Two generators, both checked for drift in CI:

```sh
make sqlc      # db/queries -> internal/infrastructure/persistence/sqlcgen
make wire      # internal/wire.go -> internal/wire_gen.go
make generate  # both
```

`wire_gen.go` and everything under `sqlcgen/` are generated. Never hand-edit
them; change the input and regenerate.

`internal/wire.go` splits the graph in two. `coreSet` takes a `*pgxpool.Pool`
as given; `InitializeAPI` adds `postgres.New` on top, and
`InitializeAPIWithPool` does not. That is what lets the integration tests
exercise the exact production wiring with only the pool substituted, instead of
maintaining a second object graph that drifts.

## Configuration

`config.Load` reads the process environment, and `DATABASE_URL` is required, so
a missing value fails at startup instead of defaulting to something surprising.
`.env` is a local convenience exported by the Makefile; the service itself never
reads a file. See `.env.example` for every knob.

## Authentication

Session cookies, not JWTs. Signing in mints 256 bits of random token, stores
its SHA-256 digest in `sessions`, and returns the token in an `HttpOnly`,
`SameSite=Lax` cookie. Nothing else in the system ever holds the plaintext, so
a database dump contains no replayable sessions, and no script on the page can
read the credential.

The pieces:

- `internal/domain/auth/entity` holds `User` and `Session`. They share a
  package because they share a bounded context, which is also what lets
  `rest/auth` obey the one-domain-package-per-REST-package rule.
- The domain holds **no cryptography**. It owns the policy (how long a password
  must be, when a session has expired); `application.PasswordHasher` and
  `application.TokenGenerator` are ports, implemented in
  `internal/infrastructure/security` with Argon2id and `crypto/rand`.
- `rest/auth.RequireSession` is the middleware every other resource is mounted
  behind. It lives in `rest/auth` rather than `rest` so the router itself
  imports no domain package at all.

Choices worth knowing about:

- **A wrong password and an unknown address are the same error.** Login also
  verifies against a decoy hash when the address is unregistered, so the two
  cost the same wall-clock time. Both together are what stop the form being an
  account-enumeration oracle.
- **`SameSite=Lax` is the CSRF defence.** The browser withholds the cookie from
  cross-site `POST`, `PATCH` and `DELETE`. It still sends it on a top-level
  `GET` navigation, which is why no `GET` in this API changes state — keep it
  that way.
- **`Secure` is forced on in production** by `config.SecureCookies`, whatever
  `SESSION_COOKIE_SECURE` says. The setting exists to relax staging, not
  production.
- **Changing a password destroys every other session** and re-issues the
  caller's cookie, so the browser making the change stays signed in and anyone
  holding a stolen cookie does not.

Registration always creates a `member`. Nothing grants `admin` over HTTP; do it
in the database, deliberately.

## API

| Method | Path | Notes |
| --- | --- | --- |
| `GET` | `/healthz` | liveness, no database dependency |
| `GET` | `/readyz` | readiness, pings the database |
| `POST` | `/v1/auth/register` | public; `201`, sets the session cookie |
| `POST` | `/v1/auth/login` | public; `401` on any bad credential |
| `POST` | `/v1/auth/logout` | `204`, clears the cookie, idempotent |
| `GET` | `/v1/auth/me` | the signed-in user |
| `PATCH` | `/v1/auth/me` | change the display name |
| `POST` | `/v1/auth/me/password` | evicts every other session |
| `POST` | `/v1/accounts` | `201` plus a `Location` header |
| `GET` | `/v1/accounts?limit=&offset=` | limit clamped to 100 |
| `GET` | `/v1/accounts/{id}` | |
| `POST` | `/v1/accounts/{id}/deposit` | |
| `POST` | `/v1/accounts/{id}/withdraw` | `422` when it would overdraw |
| `DELETE` | `/v1/accounts/{id}` | `409` unless the balance is zero |

Errors always look like `{"error": "...", "code": "..."}`. Branch on `code`;
the codes are part of the contract. Each resource package has its own
`statusFor` (`rest/account/errors.go`, `rest/auth/errors.go`,
`rest/post/errors.go`) mapping its sentinels, with a curated message per case
so no internal detail leaks: unexpected errors log in full and return a bare
`500`.

Request bodies are decoded strictly (unknown fields rejected, 1 MiB cap) and
required numeric fields use pointers, so a missing `amount_minor` is a clear
`400` rather than a silent zero.

## Quality gates

```sh
make fmt    # gofumpt + goimports
make lint   # golangci-lint, both build configurations
make check  # fmt + lint + test, the same thing CI runs
```

`.golangci.yml` uses `default: none` plus an explicit enable list, so a linter
upgrade never turns on new checks unannounced and every entry was a deliberate
choice. `nolintlint` requires an explanation on every suppression.

## Notes

- Migrations never run at service startup. Concurrent replicas would race, and
  coupling schema changes to deploys makes both harder to roll back. `migrate`
  is a separate binary, shipped in the same image.
- `/healthz` deliberately does not touch the database. If liveness depended on
  it, a brief database blip would get pods killed instead of drained.
- chi's `middleware.RealIP` is deliberately not used: it rewrites `RemoteAddr`
  from client-controlled headers and is spoofable unless a trusted proxy
  overwrites them (GHSA-3fxj-6jh8-hvhx).
- Prometheus metrics and OpenAPI generation are intentionally absent. Add them
  when you need them; `NewRouter` is the seam for the first and `db/queries` is
  unaffected by the second.
