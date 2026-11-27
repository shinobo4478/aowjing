-- A profile's cast: recurring characters, objects and places (see schema.sql).

-- name: ListCastMembers :many
SELECT * FROM cast_members
WHERE profile_id = $1
ORDER BY position, created_at;

-- Every picture of a profile's cast, grouped by the caller.
-- name: ListCastPictures :many
SELECT ci.cast_id, ci.image_id, ci.position
FROM cast_member_images ci
JOIN cast_members m ON m.id = ci.cast_id
WHERE m.profile_id = $1
ORDER BY m.position, m.created_at, ci.position;

-- name: GetCastMember :one
SELECT * FROM cast_members
WHERE id = $1 AND profile_id = $2;

-- name: CountCastMembers :one
SELECT count(*) FROM cast_members WHERE profile_id = $1;

-- name: CreateCastMember :one
INSERT INTO cast_members (profile_id, role, name, note, position)
VALUES ($1, $2, $3, $4,
        COALESCE((SELECT max(position) FROM cast_members WHERE profile_id = $1), 0) + 1)
RETURNING *;

-- name: UpdateCastMember :one
UPDATE cast_members
SET role = $3, name = $4, note = $5
WHERE id = $1 AND profile_id = $2
RETURNING *;

-- The pictures of one member, collected BEFORE it is deleted so the ones
-- nothing else uses can be removed afterwards.
-- name: ListCastMemberImageIDs :many
SELECT image_id FROM cast_member_images WHERE cast_id = $1;

-- name: DeleteCastMember :execrows
DELETE FROM cast_members
WHERE id = $1 AND profile_id = $2;

-- Takes the lowest free position (1-4), or no row when all four are taken.
-- name: AddCastImage :one
INSERT INTO cast_member_images (cast_id, image_id, position)
SELECT $1, $2, p.n
FROM generate_series(1, 4) AS p(n)
WHERE NOT EXISTS (
    SELECT 1 FROM cast_member_images c
    WHERE c.cast_id = $1 AND c.position = p.n
)
ORDER BY p.n
LIMIT 1
RETURNING position;

-- name: RemoveCastImage :execrows
DELETE FROM cast_member_images
WHERE cast_id = $1 AND image_id = $2;

-- The members a picture request ticked (only of that profile), in the cast's
-- own order, for the handler to describe in the prompt.
-- name: ListCastMembersByIDs :many
SELECT * FROM cast_members
WHERE profile_id = $1 AND id = ANY($2::uuid[])
ORDER BY position, created_at;

-- name: ListCastPicturesByCast :many
SELECT cast_id, image_id, position
FROM cast_member_images
WHERE cast_id = ANY($1::uuid[])
ORDER BY cast_id, position;
