-- name: GetStore :one
SELECT * FROM stores WHERE id = $1;

-- name: ListStores :many
SELECT * FROM stores ORDER BY created_at;

-- name: CreateStore :one
INSERT INTO stores (name, config)
VALUES ($1, $2)
RETURNING *;
