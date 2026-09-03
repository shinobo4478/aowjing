-- name: ListProfiles :many
SELECT * FROM profiles
ORDER BY created_at DESC;

-- name: GetProfile :one
SELECT * FROM profiles
WHERE id = $1;

-- name: CreateProfile :one
INSERT INTO profiles (name, niche, description, provider)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateProfile :one
UPDATE profiles
SET name = $2,
    niche = $3,
    description = $4,
    provider = $5,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteProfile :execrows
DELETE FROM profiles
WHERE id = $1;
