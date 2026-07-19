-- name: CreateAuditLog :exec
INSERT INTO audit_log (store_id, actor_id, action, entity, entity_id, detail)
VALUES ($1, $2, $3, $4, $5, $6);
