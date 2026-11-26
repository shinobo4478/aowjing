-- name: CreateKeyframe :one
INSERT INTO keyframes (profile_id, prompt, model, count, reference_count, extra_image_ids, status)
VALUES ($1, $2, $3, $4, $5, $6, 'pending')
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

-- The reference pictures of a profile as stored data, for the worker to send.
-- name: ListProfileReferenceData :many
SELECT i.content_type, i.data
FROM profile_reference_images r
JOIN images i ON i.id = r.image_id
WHERE r.profile_id = $1
ORDER BY r.position;
