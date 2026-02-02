-- name: CreateTeam :exec
INSERT INTO teams (id, name, invite_token, captain_id, is_solo, is_auto_created, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: CreateTeamReturningID :one
INSERT INTO teams (name, invite_token, captain_id, is_solo, is_auto_created, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: GetTeamByID :one
SELECT id, name, invite_token, captain_id, is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden, created_at
FROM teams
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetTeamByInviteToken :one
SELECT id, name, invite_token, captain_id, is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden, created_at
FROM teams
WHERE invite_token = $1 AND deleted_at IS NULL;

-- name: GetTeamByName :one
SELECT id, name, invite_token, captain_id, is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden, created_at
FROM teams
WHERE name = $1 AND deleted_at IS NULL;

-- name: SoftDeleteTeam :one
UPDATE teams SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: GetSoloTeamByUserID :one
SELECT t.id, t.name, t.invite_token, t.captain_id, t.is_solo, t.is_auto_created, t.is_banned, t.banned_at, t.banned_reason, t.is_hidden, t.created_at
FROM teams t
JOIN users u ON u.team_id = t.id
WHERE u.id = $1 AND t.is_solo = true AND t.deleted_at IS NULL;

-- name: CountTeamMembers :one
SELECT COUNT(*)::int FROM users WHERE team_id = $1;

-- name: BanTeam :one
UPDATE teams SET is_banned = true, banned_at = $2, banned_reason = $3
WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: UnbanTeam :one
UPDATE teams SET is_banned = false, banned_at = NULL, banned_reason = NULL
WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: SetTeamHidden :one
UPDATE teams SET is_hidden = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: UpdateTeamCaptain :exec
UPDATE teams SET captain_id = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: GetAllTeams :many
SELECT id, name, invite_token, captain_id, is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden, created_at
FROM teams
WHERE deleted_at IS NULL
ORDER BY created_at ASC;

-- name: HardDeleteTeamsBefore :exec
DELETE FROM teams WHERE deleted_at IS NOT NULL AND deleted_at < $1;
