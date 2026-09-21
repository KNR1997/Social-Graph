-- Every statement in this file is scoped by user_id, without exception. That is
-- PLAN D3: ownership is enforced here rather than in the handler, so a use case
-- that forgets a check gets "not found" instead of another user's data. A query
-- added below without a user_id predicate is a security bug, not a style slip.

-- name: InsertPerson :exec
INSERT INTO persons (
    id, user_id, is_self, full_name, email, phone, occupation,
    organization, location, notes, version, created_at, updated_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
);

-- name: GetPerson :one
SELECT id, user_id, is_self, full_name, email, phone, occupation,
       organization, location, notes, version, created_at, updated_at
FROM persons
WHERE id = $1
  AND user_id = $2;

-- GetSelfPerson loads the user's own node in the graph. Every MVP relationship
-- runs from this row (PLAN D1), so the relationship use cases resolve it on
-- create rather than trusting a from_person_id off the request body.
-- name: GetSelfPerson :one
SELECT id, user_id, is_self, full_name, email, phone, occupation,
       organization, location, notes, version, created_at, updated_at
FROM persons
WHERE user_id = $1
  AND is_self;

-- ListPersons is both the list and the search endpoint, as one query with an
-- optional term (PLAN M2). ILIKE '%...%' is served by the trigram indexes on
-- full_name and organization (PLAN D8), which is why the term is matched with a
-- substring pattern rather than a prefix.
--
-- The self person is excluded: it is the user's own node, not someone they
-- know, and leaving it in puts every user in their own contact list and makes
-- the people count off by one. GetSelfPerson is how that row is reached.
-- name: ListPersons :many
SELECT id, user_id, is_self, full_name, email, phone, occupation,
       organization, location, notes, version, created_at, updated_at
FROM persons
WHERE user_id = sqlc.arg('user_id')
  AND NOT is_self
  AND (
      sqlc.narg('search')::text IS NULL
      OR full_name ILIKE '%' || sqlc.narg('search') || '%'
      OR organization ILIKE '%' || sqlc.narg('search') || '%'
  )
ORDER BY created_at DESC, id DESC
LIMIT sqlc.arg('limit') OFFSET sqlc.arg('offset');

-- CountPersons exists so a paginated response can report a total. It repeats
-- ListPersons' predicate deliberately: the two must agree, and a count that
-- quietly filters differently is worse than no count at all.
-- name: CountPersons :one
SELECT count(*)
FROM persons
WHERE user_id = sqlc.arg('user_id')
  AND NOT is_self
  AND (
      sqlc.narg('search')::text IS NULL
      OR full_name ILIKE '%' || sqlc.narg('search') || '%'
      OR organization ILIKE '%' || sqlc.narg('search') || '%'
  );

-- UpdatePerson is the optimistic-lock write: it only matches while the row is
-- still at the version we loaded. It returns the version it assigned, so the
-- caller can report the current one rather than the stale one it wrote with; no
-- row coming back is what the repository turns into entity.ErrConflict.
--
-- user_id and is_self are not in the SET list. Ownership and the identity of
-- the self node are not editable facts, so there is no statement here that can
-- change them.
-- name: UpdatePerson :one
UPDATE persons
SET full_name    = $3,
    email        = $4,
    phone        = $5,
    occupation   = $6,
    organization = $7,
    location     = $8,
    notes        = $9,
    updated_at   = $10,
    version      = version + 1
WHERE id = $1
  AND user_id = $2
  AND version = $11
RETURNING version;

-- DeletePerson is a hard delete (PLAN D7): the row's relationships and their
-- interactions go with it by cascade, and anyone introduced through this person
-- keeps their history with met_through_person_id set to null.
--
-- The self person is excluded. Deleting it would strip the user of the node
-- every one of their relationships runs from, and the partial unique index
-- means registration cannot simply recreate it.
-- name: DeletePerson :execrows
DELETE FROM persons
WHERE id = $1
  AND user_id = $2
  AND NOT is_self;
