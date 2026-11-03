-- name: ListPlacementsByProfile :many
SELECT gp.generation_id, gp.position, p.*
FROM generation_products gp
JOIN products p ON p.id = gp.product_id
JOIN generations g ON g.id = gp.generation_id
WHERE g.profile_id = $1
ORDER BY gp.generation_id, gp.position;

-- name: ListPlacementsByGeneration :many
SELECT gp.generation_id, gp.position, p.*
FROM generation_products gp
JOIN products p ON p.id = gp.product_id
WHERE gp.generation_id = $1
ORDER BY gp.position;

-- name: ClearPlacements :exec
DELETE FROM generation_products
WHERE generation_id = $1;

-- name: AddPlacement :exec
INSERT INTO generation_products (generation_id, product_id, position)
VALUES ($1, $2, $3);

-- name: CountProductPlacements :one
SELECT count(*) FROM generation_products
WHERE product_id = $1;

-- name: GenerationExists :one
SELECT EXISTS (SELECT 1 FROM generations WHERE id = $1);
