-- name: CreateMovement :one
INSERT INTO stock_movements
  (store_id, product_id, type, qty_delta, ref_type, ref_id, reason, created_by, client_uuid)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING *;

-- name: OnHandFromLedger :one
-- Authoritative on-hand = sum of the immutable ledger.
SELECT COALESCE(SUM(qty_delta), 0)::bigint AS on_hand
FROM stock_movements
WHERE product_id = $1;

-- name: GetMovementByClientUUID :one
SELECT * FROM stock_movements
WHERE store_id = $1 AND client_uuid = $2;

-- name: ListMovementsByProduct :many
SELECT * FROM stock_movements
WHERE store_id = $1 AND product_id = $2
ORDER BY created_at DESC
LIMIT $3;
