-- name: NextBillSeq :one
-- Atomically allocate the next gap-free sequence for a store. Running inside the
-- sale transaction means a rollback also rolls back the increment → no gaps.
INSERT INTO bill_counters (store_id, next_seq)
VALUES ($1, 2)
ON CONFLICT (store_id) DO UPDATE SET next_seq = bill_counters.next_seq + 1
RETURNING next_seq - 1 AS seq;

-- name: CreateBill :one
INSERT INTO bills (
  store_id, bill_no, client_uuid, cashier_id,
  subtotal, discount, total, paid, change,
  payment_method, status, voids_bill_id, synced_at
) VALUES (
  $1, $2, $3, $4,
  $5, $6, $7, $8, $9,
  $10, $11, $12, $13
)
RETURNING *;

-- name: GetBill :one
SELECT * FROM bills WHERE id = $1 AND store_id = $2;

-- name: GetBillByClientUUID :one
SELECT * FROM bills WHERE store_id = $1 AND client_uuid = $2;

-- name: GetVoidForBill :one
SELECT * FROM bills WHERE store_id = $1 AND voids_bill_id = $2 LIMIT 1;

-- name: CreateBillItem :one
INSERT INTO bill_items (bill_id, product_id, name_snapshot, price_snapshot, qty, line_total)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: ListBillItems :many
SELECT * FROM bill_items WHERE bill_id = $1 ORDER BY id;
