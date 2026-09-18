# PLAN.md

Execution plan for the MVP defined in `README.md` (sections **MVP** and
**Roadmap Phase 1/2**), against the skeleton in `app/`.

Written 2026-09-17. The status of every checkbox lives in the README; this file
is the how, the order, and the decisions.

---

## 1. Starting point

`app/` is a working Go + React stack (chi, pgx, sqlc, goose, Wire, Postgres 18;
React 19 + Vite + shadcn). It is a generic starter: its Go module is
`go-ddd-service`, its aggregates are `account` (a bank account) and `post`
(placeholders), plus `auth` (users, sessions, cookies) which is real and stays.
`app/CLAUDE.md` documents the layering rules, which `depguard` enforces.

Nothing of the Relationship Graph domain exists yet. Two properties of the
skeleton matter more than anything else for this plan:

1. **No table is owner-scoped.** `posts` and `accounts` have no `user_id`. Every
   table this plan adds must have one, and every query must filter on it.
   Design principle 3 in the README ("the user owns the relationship data") is a
   security requirement, not a preference.
2. **`app/` is untracked in git.** Until it is committed there is no baseline to
   review changes against.

---

## 2. Scope

### In (the README MVP list, unchanged)

People CRUD and search; relationships with type, importance, how people met,
start date, preferred interaction frequency; interactions with type, notes and
history; a dashboard covering attention, recent, inactive and upcoming dates;
in-app maintenance reminders with configurable frequency.

### Out (deferred, with the phase that owns it)

- Graph traversal: mutual connections, connection paths, visualization (Phase 3).
- A percentage health score and its factor model (Phase 2, see D5).
- AI suggestions, smart scoring, interaction summaries (Phase 4).
- Email, push, calendar and contact sync, background workers (Phase 5).
- Redis. Nothing in the MVP needs a cache or a queue.

---

## 3. Decisions

Each decision is stated as taken, so work can start. The ones marked
**[confirm]** change the shape of the product rather than the code, and are
worth a yes or no before the milestone that depends on them.

**D1. The graph is stored as edges from day one, but only ego edges are written
in the MVP.** `relationships` carries `from_person_id` and `to_person_id`.
Registration creates a "self" `Person` row for the user, and every MVP
relationship runs from that self person to another person. Phase 3 then adds
person-to-person edges with no migration and no rewrite of the repository. The
cost is one extra row per user at registration; the alternative (folding
relationship attributes into `persons` now) makes Phase 3 a schema migration
plus a rewrite of every query, which is the more expensive mistake.

**D2. Person, Relationship and Interaction are three aggregates, not one.** An
interaction references its relationship by id rather than living inside the
relationship aggregate. A relationship accumulates interactions without bound,
and an aggregate that must load all of them to add one is a performance bug
waiting to be written. The consequence: any rule spanning them (overdue, health)
is a **domain service** over a relationship plus interaction statistics, not a
method on either entity.

**D3. Ownership is enforced in the repository, not the handler.** Every query in
`db/queries/` takes `user_id` in its `WHERE`, and every repository method takes
an owner id. A handler that forgets a check then gets a "not found" rather than
another user's data. A cross-owner reference (for example
`met_through_person_id` pointing at a person the caller does not own) is
rejected by a composite foreign key, not by an application lookup.

**D4. Read models are their own domain package.** The dashboard needs people,
relationships and interactions at once, and the skeleton's rule is that a REST
package imports exactly one domain package. So `internal/domain/attention/entity`
holds the read-model types (`AttentionItem`, `Summary`), populated by one
purpose-built query, and `rest/dashboard` imports that package only. Do not
solve this by letting a REST package import three domains; that rule is
load-bearing.

**D5. [confirm] The MVP ships an overdue rule, not a health score.** The README
puts relationship health in Phase 2, but the MVP dashboard ("relationships
needing attention", "inactive relationships") needs some notion of it. The MVP
compares `days_since_last_interaction` against `preferred_frequency_days` and
buckets the result into healthy / due / overdue. No percentage is shown, because
a number like "78%" implies a factor model that does not exist yet and that the
README explicitly wants to be more than a function of time. Phase 2 replaces the
bucket with a real score behind the same domain service, so no API or UI shape
changes.

**D6. [confirm] Reminders are in-app and computed on read.** "Notifications" in
the MVP means the dashboard list plus a reminders view, derived from the same
overdue rule on request. No `reminders` table, no scheduler, no email. A stored
reminder is only worth its complexity once something delivers it asynchronously,
which is Phase 5. "Configurable reminder frequency" is satisfied by the
per-relationship `preferred_frequency_days` plus a user-level grace period
(default 14 days) before a due relationship is called overdue.

**D7. Deletes are hard, with the blast radius set in the schema.** Deleting a
person cascades to its relationships and their interactions, and sets
`met_through_person_id` to null on anyone introduced through them. A person
record is the user's own note about someone, so "delete" should mean it is gone.
The introduction link survives as history precisely because losing it would
silently rewrite how the rest of the network was met.

**D8. Search starts as `pg_trgm`, not full-text search.** Names and
organizations are short strings where fuzzy substring matching is what people
expect, and `ILIKE` with a GIN trigram index carries a personal network's worth
of rows without a `tsvector` column to maintain.

**D9. The Go module is renamed to `relationshipgraph` in M1.** It touches every
import line once and costs nothing; leaving it means every file in the project
claims to be a different project.

---

## 4. Data model

Five tables, in migration order. Every one carries `user_id uuid NOT NULL
REFERENCES users (id) ON DELETE CASCADE`, plus `version`, `created_at` and
`updated_at`, matching the skeleton's conventions. `CHECK` constraints mirror
the domain invariants, as in `users`.

```
persons
  id, user_id, is_self, full_name, email, phone, occupation,
  organization, location, notes
  UNIQUE (user_id) WHERE is_self          -- one self person per user
  UNIQUE (id, user_id)                    -- target of the composite FKs below
  INDEX  gin (full_name gin_trgm_ops), gin (organization gin_trgm_ops)

relationships
  id, user_id, from_person_id, to_person_id,
  type, importance, started_at, met_through_person_id (null),
  preferred_frequency_days (null)
  FOREIGN KEY (from_person_id, user_id)        REFERENCES persons (id, user_id)
  FOREIGN KEY (to_person_id, user_id)          REFERENCES persons (id, user_id)
  FOREIGN KEY (met_through_person_id, user_id) REFERENCES persons (id, user_id)
                                               ON DELETE SET NULL
  UNIQUE (user_id, from_person_id, to_person_id)
  CHECK  (from_person_id <> to_person_id)
  CHECK  (type IN (...)), CHECK (importance IN ('low','medium','high'))
  CHECK  (preferred_frequency_days IS NULL OR preferred_frequency_days > 0)

interactions
  id, user_id, relationship_id, type, occurred_at, notes
  FOREIGN KEY (relationship_id, user_id) REFERENCES relationships (id, user_id)
                                         ON DELETE CASCADE
  CHECK  (type IN (...)), CHECK (occurred_at <= now())
  INDEX  (user_id, relationship_id, occurred_at DESC)

important_dates
  id, user_id, person_id, label, on_date, recurring (bool)
  INDEX  (user_id, on_date)

user_preferences
  user_id PRIMARY KEY, default_frequency_days, overdue_grace_days
```

The composite foreign keys are the whole of D3's enforcement: a person from
another account cannot be referenced, because the pair `(person_id, user_id)`
will not resolve.

Relationship type and interaction type are enumerated in the domain as value
objects and mirrored in a `CHECK`, not stored in a lookup table. A user-defined
type list is a real feature, but it is not in the MVP and a table now buys
nothing.

---

## 5. API surface

All under `/v1`, behind `RequireSession`. The owner comes from the session,
never from the request body.

```
POST   /v1/people                     create
GET    /v1/people?q=&limit=&offset=   list and search (D8)
GET    /v1/people/{id}                profile, with its relationship
PATCH  /v1/people/{id}                update
DELETE /v1/people/{id}                delete (D7)

POST   /v1/relationships              person_id, type, importance, started_at,
                                      met_through_person_id,
                                      preferred_frequency_days
GET    /v1/relationships
GET    /v1/relationships/{id}
PATCH  /v1/relationships/{id}         type and importance evolve over time

POST   /v1/interactions               relationship_id, type, occurred_at, notes
GET    /v1/interactions?relationship_id=&limit=&offset=

GET    /v1/dashboard                  attention, recent, inactive, upcoming
GET    /v1/reminders                  the overdue list (D6)
GET    /v1/me/preferences
PATCH  /v1/me/preferences
```

---

## 6. Milestones

Each milestone is shippable and independently reviewable. Sizes are relative:
S is a sitting, M is a day or so, L is more.

### M0. Baseline (S)

- Commit `app/` so there is a diff to review against.
- Fix the stale `.github/workflows/ci.yml` reference in `app/CLAUDE.md`, and
  either add that workflow or drop the line.
- Update the root `AGENTS.md`, which still says "no source code yet".
- **Done when** `make check` passes on a clean checkout and `git status` is clean.

### M1. Make the skeleton this project's skeleton (M)

- Rename the module to `relationshipgraph` (D9).
- Add `user_preferences`, and extend registration to create the self `Person`
  and the preferences row in the same transaction as the user (D1).
- Delete the `account` aggregate end to end: domain, application, persistence,
  `rest/account`, its queries, its migration, its tests, its Wire providers.
  Keep `post` for now as the reference pattern; it goes in M3.
- **Done when** registration yields a user with a self person, the router serves
  only `/v1/auth` and `/v1/posts`, and `make test-integration` passes.

### M2. People (L), covering README "People"

- Migration `create_persons`, with the trigram indexes.
- `internal/domain/person/entity`: `Person` with unexported fields, a `FullName`
  value object (non-blank, length-bounded), optional contact fields validated on
  set, and the `Repository` port.
- Queries, repository, `PersonService`, `rest/person` with DTOs and `statusFor`.
- Search is one query with an optional `q`, so list and search are one endpoint.
- **Done when** create, view, update, delete and search all round-trip through
  integration tests, and a second user cannot read or delete the first user's
  people. Assert that explicitly; it is the D3 regression test.

### M3. Relationships (L), covering README "Relationships"

- Migration `create_relationships`, with the composite foreign keys.
- `internal/domain/relationship/entity`: `Relationship`, plus the value objects
  `RelationshipType`, `Importance` and `InteractionFrequency`. Invariants that
  belong here and not in the service: no self edge, no duplicate edge, a start
  date that is not in the future, an introducer who is not the person themselves.
- `rest/relationship`, wired behind the session.
- Delete the `post` aggregate now that a real one has replaced it as the example.
- **Done when** the README's John/Jason/Paul journey (steps 1 to 4) can be driven
  with curl, including recording Paul as met through Jason and evolving the type
  from acquaintance to friend.

### M4. Interactions (M), covering README "Interactions"

- Migration `create_interactions`, domain package, service, `rest/interaction`.
- `InteractionType` as a value object; notes length-bounded; `occurred_at` cannot
  be in the future and is supplied by the caller rather than read from the clock,
  because people record interactions after the fact.
- **Done when** history for a relationship is paginated newest first, and an
  interaction cannot be attached to a relationship the caller does not own.

### M5. Attention and the dashboard (L), covering README "Dashboard" and "Notifications"

- Migration `create_important_dates`.
- `internal/domain/attention`: the domain service implementing D5, taking a
  relationship plus its interaction statistics and returning a bucket with the
  reason. Pure, table-driven unit tests, no database.
- `internal/domain/attention/entity`: the read-model types (D4).
- One query producing last interaction, interaction count and observed frequency
  per relationship in a single pass. Do not compute this per row in Go.
- `rest/dashboard` and `rest/reminders`, plus the preferences endpoints.
- **Done when** the four dashboard sections and the reminder list are each served
  in one round trip, and the bucketing has unit tests covering the README's own
  cases: 18 days against 30 is healthy, 61 against 30 to 45 is overdue, and a
  relationship with no preferred frequency is never called overdue.

### M6. Frontend (L)

The `web/` starter ships a dozen demo features (products, kanban, chat, users,
ai-chat) whose data is faker mocks. Strip them, keep `auth`, and add:

- `features/people` (list with search, profile, create and edit forms),
  `features/relationships`, `features/interactions` (a timeline plus a quick
  "record interaction" action), and `features/dashboard`.
- Each follows the starter's own layering: `api/types.ts` to `api/service.ts` to
  `api/queries.ts` with a key factory. Forms go through `useAppForm`. Every fetch
  goes through `src/lib/api.ts`, which is what sends the session cookie.
- Rewrite `src/config/nav-config.ts`; it drives both the sidebar and Cmd+K.
- **Done when** the README's dashboard mock is recognisable on screen and the
  full journey works against the real API: add a person, record how they were
  met, log an interaction, watch the relationship leave the attention list.

### M7. Hardening before calling the MVP done (M)

- An integration test per resource asserting cross-user isolation.
- Pagination limits enforced server side.
- `docker-compose` runs the API, the database and a built frontend together.
- One seed command producing the README's example network, so the dashboard is
  demonstrable without an afternoon of data entry.
- Decide the license. `README.md` currently says "to be determined".

---

## 7. Working agreement

- Vertical slices. A milestone touches migration, domain, persistence,
  application, REST and tests together. Do not build one layer across all
  aggregates first; `depguard` enforces the layering, so a half-built slice will
  not compile into anything demonstrable.
- Domain invariants get unit tests with no database. Repositories and handlers
  get integration tests behind the `integration` build tag. The patterns to copy
  are `internal/domain/account/entity/account_test.go` and
  `internal/infrastructure/persistence/account_repository_integration_test.go`,
  so read both before deleting `account` in M1.
- Every new mutating query keeps the optimistic-locking guard on `version` and
  maps 0 rows to `ErrConflict`.
- Tick the README's MVP checkboxes in the same commit that makes them true.

---

## 8. Open questions

1. D5 and D6: is an overdue rule plus in-app reminders enough for the MVP, or is
   a visible health percentage part of what makes it worth using?
2. Is a single-user local deployment the target, or is this hosted for other
   people from the start? It changes how much M7 has to cover.
3. `important_dates` appears in the MVP dashboard ("upcoming important dates")
   but in none of the MVP feature lists, so the checklist never asks for CRUD on
   it. M5 adds the table and the read path; the create and edit UI needs a home,
   most naturally on the person profile in M6.
