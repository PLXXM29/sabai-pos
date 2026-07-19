-- name: CreateProduct :one
INSERT INTO products (store_id, name, barcode, category, cost_price, sell_price)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetProduct :one
SELECT * FROM products WHERE id = $1 AND store_id = $2;

-- name: GetProductByBarcode :one
SELECT * FROM products
WHERE store_id = $1 AND barcode = $2 AND is_active = TRUE;

-- name: ListProductsWithStock :many
SELECT
  p.*,
  COALESCE(i.qty_on_hand, 0)::int   AS qty_on_hand,
  COALESCE(i.reorder_point, 0)::int AS reorder_point
FROM products p
LEFT JOIN inventory i ON i.product_id = p.id
WHERE p.store_id = $1 AND p.is_active = TRUE
ORDER BY p.name;

-- name: UpdateProduct :one
UPDATE products
SET name = $3, barcode = $4, category = $5, cost_price = $6, sell_price = $7
WHERE id = $1 AND store_id = $2
RETURNING *;

-- name: DeactivateProduct :execrows
UPDATE products SET is_active = FALSE
WHERE id = $1 AND store_id = $2;
