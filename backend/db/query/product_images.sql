-- Pictures of a product (see schema.sql).

-- name: ListProductPictures :many
SELECT image_id, position, label
FROM product_images
WHERE product_id = $1
ORDER BY position;

-- Every product that has pictures, with them, for the picture dialog's ticks.
-- name: ListPickableProductPictures :many
SELECT p.id AS product_id, p.name, pi.image_id, pi.position, pi.label
FROM product_images pi
JOIN products p ON p.id = pi.product_id
ORDER BY p.created_at DESC, p.id, pi.position;

-- Takes the lowest free position (1-4), or no row when all four are taken.
-- name: AddProductPicture :one
INSERT INTO product_images (product_id, image_id, position, label)
SELECT $1, $2, p.n, $3
FROM generate_series(1, 4) AS p(n)
WHERE NOT EXISTS (
    SELECT 1 FROM product_images c
    WHERE c.product_id = $1 AND c.position = p.n
)
ORDER BY p.n
LIMIT 1
RETURNING position;

-- name: UpdateProductPictureLabel :execrows
UPDATE product_images
SET label = $3
WHERE product_id = $1 AND image_id = $2;

-- name: RemoveProductPicture :execrows
DELETE FROM product_images
WHERE product_id = $1 AND image_id = $2;

-- The pictures of one product, collected BEFORE it is deleted so the ones
-- nothing else uses can be removed afterwards.
-- name: ListProductImageIDs :many
SELECT image_id FROM product_images WHERE product_id = $1;

-- The product pictures a picture request ticked: which product each belongs to
-- and its label, for the prompt. Pictures that are not a product's are simply
-- missing from the answer (the handler compares the counts).
-- name: ListProductPicturesByImageIDs :many
SELECT pi.image_id, pi.product_id, p.name, pi.label
FROM product_images pi
JOIN products p ON p.id = pi.product_id
WHERE pi.image_id = ANY($1::uuid[]);
