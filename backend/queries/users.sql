-- name: CreateUser :one
INSERT INTO users (store_id, username, password_hash, role, pin_hash)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetUserByID :one
SELECT * FROM users WHERE id = $1;

-- name: GetActiveUserByUsername :one
-- Single-tenant login: username resolves to one active user. When true
-- multi-tenant login lands, add store scoping / a store selector here.
SELECT * FROM users
WHERE username = $1 AND is_active = TRUE
LIMIT 1;

-- name: CountUsersInStore :one
SELECT count(*) FROM users WHERE store_id = $1;

-- name: SetUserActive :exec
UPDATE users SET is_active = $2 WHERE id = $1;

-- name: UpdateUserPassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;
