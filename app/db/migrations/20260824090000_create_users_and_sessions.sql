-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id            uuid        PRIMARY KEY,
    email         text        NOT NULL,
    name          text        NOT NULL,
    password_hash text        NOT NULL,
    role          text        NOT NULL DEFAULT 'member',
    version       bigint      NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),

    -- These CHECKs deliberately mirror the aggregate's invariants. The domain is
    -- the single source of truth; this is defence in depth for anything that
    -- reaches the table without going through the application (backfills, psql).
    CONSTRAINT users_email_normalised   CHECK (email = lower(btrim(email))),
    CONSTRAINT users_email_shaped       CHECK (email ~ '^[^@[:space:]]+@[^@[:space:]]+$'),
    CONSTRAINT users_email_max_len      CHECK (length(email) <= 254),
    CONSTRAINT users_name_not_blank     CHECK (length(btrim(name)) > 0),
    CONSTRAINT users_name_max_len       CHECK (length(name) <= 200),
    CONSTRAINT users_role_known         CHECK (role IN ('member', 'admin')),
    CONSTRAINT users_version_positive   CHECK (version >= 1),

    -- The last line of defence against a plaintext password reaching the table.
    -- Nothing in the application can put one here, but a backfill script written
    -- in a hurry can, and this turns that into a failed INSERT instead of a
    -- silent breach.
    CONSTRAINT users_password_hashed    CHECK (password_hash LIKE '$argon2id$%')
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Email is the identity, so uniqueness is a constraint rather than a
-- convention. The registration use case checks first for a clean error message;
-- this is what actually holds when two registrations race.
CREATE UNIQUE INDEX users_email_key ON users (email);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TABLE sessions (
    id         uuid        PRIMARY KEY,
    user_id    uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash text        NOT NULL,
    issued_at  timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,

    -- A 64-character lowercase hex string: the SHA-256 digest of the token. The
    -- token itself is never stored, and this constraint is what makes that
    -- visible at the schema level rather than only in a comment.
    CONSTRAINT sessions_token_hash_sha256 CHECK (token_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT sessions_expires_after_issue CHECK (expires_at > issued_at)
);
-- +goose StatementEnd

-- +goose StatementBegin
-- Sessions are looked up by digest on every authenticated request, so this
-- index is on the hot path for the whole API.
CREATE UNIQUE INDEX sessions_token_hash_key ON sessions (token_hash);
-- +goose StatementEnd

-- +goose StatementBegin
-- Supports both "sign out everywhere" and the expiry sweep.
CREATE INDEX sessions_user_id_idx ON sessions (user_id);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX sessions_expires_at_idx ON sessions (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE sessions;
-- +goose StatementEnd

-- +goose StatementBegin
DROP TABLE users;
-- +goose StatementEnd
