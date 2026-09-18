-- name: InsertSession :exec
INSERT INTO sessions (
    id, user_id, token_hash, issued_at, expires_at
) VALUES (
    $1, $2, $3, $4, $5
);

-- GetSessionByTokenHash filters on expiry in the query as well as in the
-- aggregate. Doing it here means an expired session cannot be loaded at all,
-- which keeps a bug in a future caller from turning into an authentication
-- bypass; the aggregate's own check then covers the row that expires between
-- this read and that check.
-- name: GetSessionByTokenHash :one
SELECT id, user_id, token_hash, issued_at, expires_at
FROM sessions
WHERE token_hash = $1
  AND expires_at > $2;

-- name: DeleteSession :exec
DELETE FROM sessions
WHERE id = $1;

-- name: DeleteSessionsByUser :exec
DELETE FROM sessions
WHERE user_id = $1;

-- name: DeleteExpiredSessions :execrows
DELETE FROM sessions
WHERE expires_at <= $1;
