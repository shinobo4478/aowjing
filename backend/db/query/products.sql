-- name: ListProducts :many
-- Optional name search: pass '' for everything.
SELECT p.*,
       (SELECT count(*) FROM generation_products gp WHERE gp.product_id = p.id) AS used_count
FROM products p
WHERE $1::text = '' OR p.name ILIKE '%' || $1::text || '%'
ORDER BY p.created_at DESC;

-- name: GetProduct :one
SELECT * FROM products
WHERE id = $1;

-- name: FindProductByShopeeIDs :one
SELECT * FROM products
WHERE shopee_shop_id = $1 AND shopee_item_id = $2;

-- name: CreateProduct :one
INSERT INTO products (name, shopee_url, shopee_shop_id, shopee_item_id, affiliate_url, price_thb, note)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateProduct :one
UPDATE products
SET name = $2,
    shopee_url = $3,
    shopee_shop_id = $4,
    shopee_item_id = $5,
    affiliate_url = $6,
    price_thb = $7,
    note = $8,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteProduct :execrows
DELETE FROM products
WHERE id = $1;

-- name: ListProductUsage :many
-- The clips a product is pinned to, newest first.
SELECT g.id AS generation_id, g.profile_id, pr.name AS profile_name,
       pt.name AS template_name, gp.position, g.created_at
FROM generation_products gp
JOIN generations g ON g.id = gp.generation_id
JOIN profiles pr ON pr.id = g.profile_id
LEFT JOIN prompt_templates pt ON pt.id = g.prompt_template_id
WHERE gp.product_id = $1
ORDER BY g.created_at DESC;
