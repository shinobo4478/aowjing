-- name: ListProfiles :many
SELECT * FROM profiles
ORDER BY created_at DESC;

-- name: GetProfile :one
SELECT * FROM profiles
WHERE id = $1;

-- name: CreateProfile :one
INSERT INTO profiles (name, niche, description, provider, video_style, video_model, video_audio, look_notes)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: UpdateProfile :one
UPDATE profiles
SET name = $2,
    niche = $3,
    description = $4,
    provider = $5,
    video_style = $6,
    video_model = $7,
    video_audio = $8,
    look_notes = $9,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteProfile :execrows
DELETE FROM profiles
WHERE id = $1;
