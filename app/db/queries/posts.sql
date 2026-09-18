-- name: InsertPost :exec
INSERT INTO posts (
    id, name, version, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5
);

-- name: GetPost :one
SELECT id, name, version, created_at, updated_at
FROM posts
WHERE id = $1;

-- UpdatePost is the optimistic-lock write: it only matches while the row is
-- still at the version we loaded, and returns the affected row count so the
-- repository can turn 0 into entity.ErrConflict.
-- name: UpdatePost :execrows
UPDATE posts
SET name       = $2,
    updated_at = $3,
    version    = version + 1
WHERE id = $1
  AND version = $4;

-- name: ListPosts :many
SELECT id, name, version, created_at, updated_at
FROM posts
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;
