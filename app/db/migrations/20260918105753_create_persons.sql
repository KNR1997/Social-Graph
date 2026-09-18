-- +goose Up

-- +goose StatementBegin
-- pg_trgm powers the trigram indexes below, which is what makes the
-- ILIKE '%...%' search behind GET /v1/people fast (PLAN D8). It is a
-- database-wide extension rather than a per-table object, so it is created here
-- only because persons is the first table that needs it.
CREATE EXTENSION IF NOT EXISTS pg_trgm;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE persons (
    id           uuid        PRIMARY KEY,

    -- Every row in this schema is owned by exactly one user, and every query
    -- filters on this column (PLAN D3). ON DELETE CASCADE means closing an
    -- account takes the whole personal network with it, rather than leaving
    -- orphaned rows nobody can reach or delete.
    user_id      uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,

    -- The user's own node in the graph, created at registration. Every MVP
    -- relationship runs from this person to another one (PLAN D1).
    is_self      boolean     NOT NULL DEFAULT false,

    full_name    text        NOT NULL,

    -- Contact details are optional. A person met at a conference may have a
    -- name and nothing else, and forcing a placeholder email would put junk in
    -- the column that every later feature has to defend against.
    email        text,
    phone        text,
    occupation   text,
    organization text,
    location     text,
    notes        text,

    -- Optimistic locking. Reads return this value, mutating statements require
    -- it in their WHERE clause and increment it, so a concurrent edit updates
    -- zero rows instead of silently overwriting the other one. The repository
    -- maps that zero to ErrConflict (PLAN section 7).
    version      bigint      NOT NULL DEFAULT 1,

    created_at   timestamptz NOT NULL DEFAULT now(),
    updated_at   timestamptz NOT NULL DEFAULT now(),

    -- These CHECKs deliberately mirror the aggregate's invariants. The domain is
    -- the single source of truth; this is defence in depth for anything that
    -- reaches the table without going through the application (backfills, psql).
    --
    -- A CHECK is evaluated per row and is satisfied when it evaluates to NULL,
    -- so every constraint on a nullable column below is automatically true when
    -- that column is NULL. That is the intended behaviour for optional fields.
    CONSTRAINT persons_full_name_not_blank CHECK (length(btrim(full_name)) > 0),
    CONSTRAINT persons_full_name_max_len   CHECK (length(full_name) <= 200),

    -- Matches the shape and normalisation rules already enforced on users.email,
    -- so the same address is stored identically wherever it appears.
    CONSTRAINT persons_email_normalised    CHECK (email = lower(btrim(email))),
    CONSTRAINT persons_email_shaped        CHECK (email ~ '^[^@[:space:]]+@[^@[:space:]]+$'),
    CONSTRAINT persons_email_max_len       CHECK (length(email) <= 254),

    -- The repository stores NULL for an absent field, never an empty string.
    -- These constraints keep that the only representation: without them a
    -- backfill could introduce '' and every later query would have to treat two
    -- values as meaning "not recorded".
    CONSTRAINT persons_email_not_blank        CHECK (email        IS NULL OR length(btrim(email))        > 0),
    CONSTRAINT persons_phone_not_blank        CHECK (phone        IS NULL OR length(btrim(phone))        > 0),
    CONSTRAINT persons_occupation_not_blank   CHECK (occupation   IS NULL OR length(btrim(occupation))   > 0),
    CONSTRAINT persons_organization_not_blank CHECK (organization IS NULL OR length(btrim(organization)) > 0),
    CONSTRAINT persons_location_not_blank     CHECK (location     IS NULL OR length(btrim(location))     > 0),
    CONSTRAINT persons_notes_not_blank        CHECK (notes        IS NULL OR length(btrim(notes))        > 0),

    -- Deliberately not a format check. Phone numbers are entered by humans in
    -- every conceivable shape, and a regex here would reject valid input; the
    -- only real invariant is that the field is not an essay.
    CONSTRAINT persons_phone_max_len       CHECK (length(phone) <= 50),

    CONSTRAINT persons_occupation_max_len   CHECK (length(occupation) <= 200),
    CONSTRAINT persons_organization_max_len CHECK (length(organization) <= 200),
    CONSTRAINT persons_location_max_len     CHECK (length(location) <= 200),
    CONSTRAINT persons_notes_max_len        CHECK (length(notes) <= 10000),

    CONSTRAINT persons_version_positive    CHECK (version >= 1)
);
-- +goose StatementEnd

-- +goose StatementBegin
-- "One self person per user" spans rows, so it cannot be a CHECK; a partial
-- unique index is how a conditional cross-row rule is expressed. The
-- registration transaction depends on this holding even if two registrations
-- race (PLAN D1).
CREATE UNIQUE INDEX persons_one_self_per_user_key ON persons (user_id) WHERE is_self;
-- +goose StatementEnd

-- +goose StatementBegin
-- Redundant as a uniqueness claim, since id is already the primary key. It
-- exists because a foreign key must point at a unique constraint covering
-- exactly its columns, and relationships and important_dates reference
-- (person_id, user_id) as a pair. That composite FK is the whole of PLAN D3's
-- enforcement: a person belonging to another account cannot be referenced,
-- because the pair will not resolve. Adding it now costs one index; adding it
-- in M3 means altering a populated table.
CREATE UNIQUE INDEX persons_id_user_id_key ON persons (id, user_id);
-- +goose StatementEnd

-- +goose StatementBegin
-- Trigram indexes for fuzzy substring search (PLAN D8). gin_trgm_ops tells the
-- GIN index to store the 3-character runs of the text rather than the string as
-- one value, which is what lets ILIKE '%jas%' use an index at all: a B-tree can
-- only answer prefix matches. It also means near-misses like Jonathan/Johnathan
-- still match, since they share most of their trigrams.
CREATE INDEX persons_full_name_trgm_idx ON persons USING gin (full_name gin_trgm_ops);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX persons_organization_trgm_idx ON persons USING gin (organization gin_trgm_ops);
-- +goose StatementEnd

-- +goose StatementBegin
-- Every list query is scoped to one user and paginated newest first, so the
-- owner leads the index. Matches the keyset pagination pattern used by accounts
-- and posts.
CREATE INDEX persons_user_id_created_at_id_idx ON persons (user_id, created_at DESC, id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Dropping the table drops its indexes with it. pg_trgm is deliberately left in
-- place: it is database-wide, other tables may come to rely on it, and dropping
-- an extension another migration installed is not this migration's business.
DROP TABLE persons;
-- +goose StatementEnd
