-- Reports exclude voided sales: a completed bill counts only if no void bill
-- references it. Profit is approximate (uses current product cost).

-- name: SalesTotals :one
SELECT
  COALESCE(SUM(total), 0)::bigint AS sales,
  COUNT(*)::bigint               AS bills
FROM bills b
WHERE b.store_id = $1 AND b.status = 'completed'
  AND b.created_at >= $2 AND b.created_at < $3
  AND NOT EXISTS (SELECT 1 FROM bills v WHERE v.voids_bill_id = b.id AND v.status = 'void');

-- name: ProfitApprox :one
SELECT COALESCE(SUM(bi.line_total - bi.qty * p.cost_price), 0)::bigint AS profit
FROM bills b
JOIN bill_items bi ON bi.bill_id = b.id
JOIN products p    ON p.id = bi.product_id
WHERE b.store_id = $1 AND b.status = 'completed'
  AND b.created_at >= $2 AND b.created_at < $3
  AND NOT EXISTS (SELECT 1 FROM bills v WHERE v.voids_bill_id = b.id AND v.status = 'void');

-- name: LowStockCount :one
SELECT COUNT(*)::bigint AS low
FROM inventory
WHERE store_id = $1 AND qty_on_hand <= reorder_point;

-- name: TopProducts :many
SELECT p.id, p.name, COALESCE(SUM(-sm.qty_delta), 0)::bigint AS qty_sold
FROM products p
JOIN stock_movements sm ON sm.product_id = p.id AND sm.type = 'sale'
WHERE p.store_id = $1
GROUP BY p.id, p.name
ORDER BY qty_sold DESC
LIMIT $2;

-- name: DailySales :many
SELECT
  (b.created_at AT TIME ZONE 'Asia/Bangkok')::date AS day,
  COALESCE(SUM(b.total), 0)::bigint                AS sales
FROM bills b
WHERE b.store_id = $1 AND b.status = 'completed' AND b.created_at >= $2
  AND NOT EXISTS (SELECT 1 FROM bills v WHERE v.voids_bill_id = b.id AND v.status = 'void')
GROUP BY day
ORDER BY day;
