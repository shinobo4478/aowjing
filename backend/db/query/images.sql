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

-- Removes an image once no clip uses it (called after a clip is deleted).
-- name: DeleteImageIfUnused :exec
DELETE FROM images i
WHERE i.id = $1
  AND NOT EXISTS (SELECT 1 FROM generations g WHERE g.image_id = i.id);
