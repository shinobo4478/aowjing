-- name: CreateYouTubeUpload :one
INSERT INTO youtube_uploads (generation_id, channel_id, youtube_video_id, privacy, title)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListYouTubeUploadsByProfile :many
SELECT yu.* FROM youtube_uploads yu
JOIN generations g ON g.id = yu.generation_id
WHERE g.profile_id = $1
ORDER BY yu.created_at DESC;
