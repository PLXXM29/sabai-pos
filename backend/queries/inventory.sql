-- name: CreateInventory :one
INSERT INTO inventory (product_id, store_id, qty_on_hand, reorder_point)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetInventory :one
SELECT * FROM inventory WHERE product_id = $1;

-- name: GetInventoryForUpdate :one
-- Locks the row so a concurrent sale of the same product can't oversell.
SELECT * FROM inventory WHERE product_id = $1 FOR UPDATE;

-- name: AddInventoryQty :one
-- Applies a delta to the cached on-hand (source of truth is stock_movements).
UPDATE inventory
SET qty_on_hand = qty_on_hand + $2
WHERE product_id = $1
RETURNING *;

-- name: SetReorderPoint :exec
UPDATE inventory SET reorder_point = $2 WHERE product_id = $1;
