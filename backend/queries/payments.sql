-- name: CreatePayment :one
INSERT INTO payments (store_id, bill_client_uuid, amount, expires_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetPayment :one
SELECT * FROM payments WHERE id = $1 AND store_id = $2;

-- name: MatchPendingPaymentForUpdate :one
-- Oldest pending intent for this exact amount that hasn't expired.
SELECT * FROM payments
WHERE status = 'pending' AND amount = $1 AND expires_at > now()
ORDER BY created_at
LIMIT 1
FOR UPDATE SKIP LOCKED;

-- name: MarkPaymentPaid :exec
UPDATE payments
SET status = 'paid', paid_at = now(), ref = $2, raw_note = $3
WHERE id = $1;

-- name: CancelPayment :exec
UPDATE payments SET status = 'cancelled'
WHERE id = $1 AND store_id = $2 AND status = 'pending';
