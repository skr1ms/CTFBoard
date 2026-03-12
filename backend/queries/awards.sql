-- name: CreateAward :exec
INSERT INTO awards (id, team_id, value, description, created_by, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetAwardsByTeamID :many
SELECT id, team_id, value, description, created_by, created_at
FROM awards
WHERE team_id = $1 AND banned_team_id IS NULL
ORDER BY created_at DESC;

-- name: GetTeamTotalAwards :one
SELECT COALESCE(SUM(value), 0)::int FROM awards WHERE team_id = $1 AND banned_team_id IS NULL;

-- name: GetAllAwards :many
SELECT id, team_id, value, description, created_by, created_at
FROM awards
WHERE banned_team_id IS NULL
ORDER BY created_at ASC;

-- name: GetAwardsForBackup :many
SELECT id, team_id, value, description, created_by, created_at, banned_team_id
FROM awards
ORDER BY created_at ASC;

-- name: GetAwardByID :one
SELECT id, team_id, value, description, created_by, created_at
FROM awards
WHERE id = $1 AND banned_team_id IS NULL;

-- name: DeleteAward :exec
DELETE FROM awards WHERE id = $1;

-- name: DeleteAwardsByTeamID :exec
DELETE FROM awards WHERE team_id = $1;

-- name: SoftBanAwardsByTeamID :exec
UPDATE awards SET banned_team_id = team_id WHERE team_id = $1 AND banned_team_id IS NULL;

-- name: RestoreAwardsByBannedTeamID :exec
UPDATE awards SET banned_team_id = NULL WHERE banned_team_id = $1;
