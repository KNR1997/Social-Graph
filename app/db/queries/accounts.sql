-- name: InsertAccount :exec
INSERT INTO accounts (
    id, owner, balance_minor, currency, status, version, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: GetAccount :one
SELECT id, owner, balance_minor, currency, status, version, created_at, updated_at
FROM accounts
WHERE id = $1;

-- UpdateAccount is the optimistic-lock write: it only matches while the row is
-- still at the version we loaded, and returns the affected row count so the
-- repository can turn 0 into entity.ErrConflict.
-- name: UpdateAccount :execrows
UPDATE accounts
SET balance_minor = $2,
    status        = $3,
    updated_at    = $4,
    version       = version + 1
WHERE id = $1
  AND version = $5;

-- name: ListAccounts :many
SELECT id, owner, balance_minor, currency, status, version, created_at, updated_at
FROM accounts
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;
