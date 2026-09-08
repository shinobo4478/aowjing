-- name: GetYouTubeAccount :one
SELECT * FROM youtube_accounts
WHERE channel_id = $1;

-- name: ListYouTubeAccountsByProfile :many
SELECT ya.* FROM youtube_accounts ya
JOIN channels c ON c.id = ya.channel_id
WHERE c.profile_id = $1;

-- name: UpsertYouTubeAccount :one
INSERT INTO youtube_accounts (
    channel_id, youtube_channel_id, youtube_title,
    access_token, refresh_token, token_expiry, scope, updated_at
)
VALUES ($1, $2, $3, $4, $5, $6, $7, now())
ON CONFLICT (channel_id) DO UPDATE
SET youtube_channel_id = EXCLUDED.youtube_channel_id,
    youtube_title      = EXCLUDED.youtube_title,
    access_token       = EXCLUDED.access_token,
    -- Keep the stored refresh token if this exchange didn't return one.
    refresh_token      = COALESCE(NULLIF(EXCLUDED.refresh_token, ''), youtube_accounts.refresh_token),
    token_expiry       = EXCLUDED.token_expiry,
    scope              = EXCLUDED.scope,
    updated_at         = now()
RETURNING *;

-- name: UpdateYouTubeTokens :exec
UPDATE youtube_accounts
SET access_token = $2,
    token_expiry = $3,
    updated_at   = now()
WHERE channel_id = $1;

-- name: DeleteYouTubeAccount :execrows
DELETE FROM youtube_accounts
WHERE channel_id = $1;
