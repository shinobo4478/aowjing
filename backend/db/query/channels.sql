-- name: ListChannelsByProfile :many
SELECT * FROM channels
WHERE profile_id = $1
ORDER BY created_at DESC;

-- name: GetChannel :one
SELECT * FROM channels
WHERE id = $1;

-- name: CreateChannel :one
INSERT INTO channels (profile_id, name, platform, handle, description)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: UpdateChannel :one
UPDATE channels
SET name = $2,
    platform = $3,
    handle = $4,
    description = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteChannel :execrows
DELETE FROM channels
WHERE id = $1;

-- name: ProfileExists :one
SELECT EXISTS (SELECT 1 FROM profiles WHERE id = $1);
