-- name: CreateTeamAuditLog :exec
INSERT INTO team_audit_log (id, team_id, user_id, action, details, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetLatestTeamAuditLogByTeamIDAndAction :one
SELECT id, team_id, user_id, action, details, created_at
FROM team_audit_log
WHERE team_id = $1 AND action = $2
ORDER BY created_at DESC
LIMIT 1;
