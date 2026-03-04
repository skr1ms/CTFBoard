-- name: CreateTeamAuditLog :exec
INSERT INTO team_audit_log (id, team_id, user_id, action, details, created_at)
VALUES ($1, $2, $3, $4, $5, $6);
