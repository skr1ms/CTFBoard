-- name: CreateAward :exec
INSERT INTO awards (id, team_id, value, description, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetAwardsByTeamID :many
SELECT id, team_id, value, description, created_by, created_at
FROM awards
WHERE team_id = $1
ORDER BY created_at DESC;

-- name: GetTeamTotalAwards :one
SELECT COALESCE(SUM(value), 0)::int FROM awards WHERE team_id = $1;

-- name: GetAllAwards :many
SELECT id, team_id, value, description, created_by, created_at
FROM awards
ORDER BY created_at ASC;
