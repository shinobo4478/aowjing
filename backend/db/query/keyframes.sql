-- name: CreateKeyframe :one
INSERT INTO keyframes (profile_id, prompt, model, count, reference_count, extra_image_ids, cast_image_ids, context, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending')
RETURNING *;

-- Pictures by id, for the worker to send as references. Order is the caller's
-- business (it sorts by the ids it asked for).
-- name: ListImagesByIDs :many
SELECT id, content_type, data
FROM images
WHERE id = ANY($1::uuid[]);

-- name: GetKeyframe :one
SELECT * FROM keyframes
WHERE id = $1;

-- name: FinishKeyframe :exec
UPDATE keyframes
SET status = $2, error = $3
WHERE id = $1;

-- name: AddKeyframeImage :exec
INSERT INTO keyframe_images (keyframe_id, image_id, position)
VALUES ($1, $2, $3);

-- name: ListKeyframeImages :many
SELECT image_id, position
FROM keyframe_images
WHERE keyframe_id = $1
ORDER BY position;

-- name: DeleteKeyframe :execrows
DELETE FROM keyframes
WHERE id = $1;

-- The newest request kept for a place (a shot's start or end picture) that has
-- not failed, so the dialog can show it again when it is reopened.
-- name: GetLatestKeyframeByContext :one
SELECT * FROM keyframes
WHERE profile_id = $1 AND context = $2 AND status <> 'failed'
ORDER BY created_at DESC
LIMIT 1;

-- Older requests kept for the same place, replaced by a new one.
-- name: ListOlderKeyframesByContext :many
SELECT id FROM keyframes
WHERE profile_id = $1 AND context = $2 AND id <> $3;

-- Kept requests whose shot is gone ('shot:<id>:start|end'), to be swept away.
-- name: ListOrphanKeyframes :many
SELECT k.id FROM keyframes k
WHERE k.context LIKE 'shot:%'
  AND NOT EXISTS (SELECT 1 FROM video_shots s WHERE s.id::text = split_part(k.context, ':', 2));
