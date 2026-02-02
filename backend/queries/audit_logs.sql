-- name: CreateAuditLog :one
INSERT INTO audit_logs (user_id, action, entity_type, entity_id, ip, details)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, created_at;
