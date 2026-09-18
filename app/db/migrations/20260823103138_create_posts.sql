-- +goose Up
-- +goose StatementBegin
CREATE TABLE posts (
    id              uuid        PRIMARY KEY,
    name            text        NOT NULL,
    version         bigint      NOT NULL DEFAULT 1,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now(),

    CONSTRAINT posts_name_not_blank  CHECK (length(btrim(name)) > 0),
    CONSTRAINT posts_version_positive CHECK (version >= 1)
);
-- +goose StatementEnd

-- +goose StatementBegin
CREATE INDEX posts_created_at_id_idx ON posts (created_at DESC, id DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE posts;
-- +goose StatementEnd
