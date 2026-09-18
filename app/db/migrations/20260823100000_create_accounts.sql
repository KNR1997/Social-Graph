-- +goose Up
-- +goose StatementBegin
CREATE TABLE accounts (
    id            uuid        PRIMARY KEY,
    owner         text        NOT NULL,
    balance_minor bigint      NOT NULL DEFAULT 0,
    currency      text        NOT NULL,
    status        text        NOT NULL,
    version       bigint      NOT NULL DEFAULT 1,
    created_at    timestamptz NOT NULL DEFAULT now(),
    updated_at    timestamptz NOT NULL DEFAULT now(),

    -- These CHECKs deliberately mirror the aggregate's invariants. The domain is
    -- the single source of truth; this is defence in depth for anything that
    -- reaches the table without going through the application (backfills, psql).
    CONSTRAINT accounts_owner_not_blank  CHECK (length(btrim(owner)) > 0),
    CONSTRAINT accounts_owner_max_len    CHECK (length(owner) <= 200),
    CONSTRAINT accounts_balance_not_neg  CHECK (balance_minor >= 0),
    CONSTRAINT accounts_currency_iso4217 CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT accounts_status_known     CHECK (status IN ('active', 'frozen', 'closed')),
    CONSTRAINT accounts_version_positive CHECK (version >= 1)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX accounts_created_at_id_idx ON accounts (created_at DESC, id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE accounts;
-- +goose StatementEnd
