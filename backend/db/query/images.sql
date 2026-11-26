-- name: CreateImage :one
INSERT INTO images (content_type, size_bytes, data)
VALUES ($1, $2, $3)
RETURNING id, content_type, size_bytes, created_at;

-- name: ImageExists :one
SELECT EXISTS (SELECT 1 FROM images WHERE id = $1);

-- name: GetImage :one
SELECT id, content_type, size_bytes, data
FROM images
WHERE id = $1;

-- Removes an image once nothing uses it: not a clip's starting or ending image,
-- not a profile's reference picture, not a keyframe candidate, not a shot's
-- starting or ending picture. Called after the thing that used it is deleted.
-- (The cascades on profile_reference_images and keyframe_images would otherwise
-- silently drop those links.)
-- name: DeleteImageIfUnused :exec
DELETE FROM images i
WHERE i.id = $1
  AND NOT EXISTS (SELECT 1 FROM generations g WHERE g.image_id = i.id OR g.end_image_id = i.id)
  AND NOT EXISTS (SELECT 1 FROM profile_reference_images r WHERE r.image_id = i.id)
  AND NOT EXISTS (SELECT 1 FROM keyframe_images k WHERE k.image_id = i.id)
  AND NOT EXISTS (SELECT 1 FROM video_shots s WHERE s.image_id = i.id OR s.end_image_id = i.id);

-- Every picture a profile owns, collected BEFORE the profile is deleted so the
-- ones nothing else uses can be removed afterwards.
-- name: ListProfileImageIDs :many
SELECT g.image_id AS image_id FROM generations g
WHERE g.profile_id = $1 AND g.image_id IS NOT NULL
UNION
SELECT g.end_image_id FROM generations g
WHERE g.profile_id = $1 AND g.end_image_id IS NOT NULL
UNION
SELECT r.image_id FROM profile_reference_images r WHERE r.profile_id = $1
UNION
SELECT k.image_id
FROM keyframe_images k
JOIN keyframes f ON f.id = k.keyframe_id
WHERE f.profile_id = $1
UNION
SELECT s.image_id
FROM video_shots s
JOIN videos v ON v.id = s.video_id
WHERE v.profile_id = $1 AND s.image_id IS NOT NULL
UNION
SELECT s.end_image_id
FROM video_shots s
JOIN videos v ON v.id = s.video_id
WHERE v.profile_id = $1 AND s.end_image_id IS NOT NULL;
