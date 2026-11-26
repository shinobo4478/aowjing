-- Reference pictures that give a profile its own look (see schema.sql).

-- name: ListProfileReferences :many
SELECT image_id, position
FROM profile_reference_images
WHERE profile_id = $1
ORDER BY position;

-- Takes the lowest free position (1-4), or no row when all four are taken.
-- name: AddProfileReference :one
INSERT INTO profile_reference_images (profile_id, image_id, position)
SELECT $1, $2, p.n
FROM generate_series(1, 4) AS p(n)
WHERE NOT EXISTS (
    SELECT 1 FROM profile_reference_images r
    WHERE r.profile_id = $1 AND r.position = p.n
)
ORDER BY p.n
LIMIT 1
RETURNING position;

-- name: RemoveProfileReference :execrows
DELETE FROM profile_reference_images
WHERE profile_id = $1 AND image_id = $2;
