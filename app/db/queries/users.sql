-- name: InsertUser :exec
INSERT INTO users (
    id, email, name, password_hash, role, version, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
);

-- name: GetUser :one
SELECT id, email, name, password_hash, role, version, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, email, name, password_hash, role, version, created_at, updated_at
FROM users
WHERE email = $1;

-- UpdateUser is the optimistic-lock write: it only matches while the row is
-- still at the version we loaded, and returns the affected row count so the
-- repository can turn 0 into entity.ErrConflict.
-- name: UpdateUser :execrows
UPDATE users
SET name          = $2,
    password_hash = $3,
    updated_at    = $4,
    version       = version + 1
WHERE id = $1
  AND version = $5;
