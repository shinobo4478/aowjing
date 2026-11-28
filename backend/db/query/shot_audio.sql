-- The voice-over of a shot (see schema.sql).

-- name: UpsertShotAudio :exec
INSERT INTO shot_audio (shot_id, voice, text, content_type, size_bytes, duration_ms, data)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (shot_id) DO UPDATE
SET voice = EXCLUDED.voice,
    text = EXCLUDED.text,
    content_type = EXCLUDED.content_type,
    size_bytes = EXCLUDED.size_bytes,
    duration_ms = EXCLUDED.duration_ms,
    data = EXCLUDED.data,
    created_at = now();

-- The audio itself, to play or download.
-- name: GetShotAudioData :one
SELECT content_type, data, created_at
FROM shot_audio
WHERE shot_id = $1;

-- What a video's shots have been spoken with, without the audio bytes.
-- name: ListShotAudioMeta :many
SELECT a.shot_id, a.voice, a.text, a.size_bytes, a.duration_ms, a.created_at
FROM shot_audio a
JOIN video_shots s ON s.id = a.shot_id
WHERE s.video_id = $1;

-- name: DeleteShotAudio :execrows
DELETE FROM shot_audio
WHERE shot_id = $1;
