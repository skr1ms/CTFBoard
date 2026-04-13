-- name: CreateTeamReturningID :one
INSERT INTO teams (name, invite_token, captain_id, is_solo, is_auto_created, created_at, invite_token_expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: GetTeamByID :one
SELECT id, name, invite_token, invite_token_expires_at, captain_id, bracket_id, is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden, avatar_url, created_at
FROM teams
WHERE id = $1 AND deleted_at IS NULL;

-- name: GetTeamByInviteToken :one
SELECT id, name, invite_token, invite_token_expires_at, captain_id, bracket_id, is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden, avatar_url, created_at
FROM teams
WHERE invite_token = $1 AND deleted_at IS NULL;

-- name: GetTeamByName :one
SELECT id, name, invite_token, invite_token_expires_at, captain_id, bracket_id, is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden, avatar_url, created_at
FROM teams
WHERE name = $1 AND deleted_at IS NULL;

-- name: SoftDeleteTeam :one
UPDATE teams SET deleted_at = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: GetSoloTeamByUserID :one
SELECT t.id, t.name, t.invite_token, t.invite_token_expires_at, t.captain_id, t.bracket_id, t.is_solo, t.is_auto_created, t.is_banned, t.banned_at, t.banned_reason, t.is_hidden, t.avatar_url, t.created_at
FROM teams t
JOIN users u ON u.team_id = t.id
WHERE u.id = $1 AND t.is_solo = true AND t.deleted_at IS NULL;

-- name: CountTeamMembers :one
SELECT COUNT(*)::int FROM users WHERE team_id = $1;

-- name: CountActiveTeams :one
SELECT COUNT(*)::int
FROM teams
WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false;

-- name: BanTeam :one
UPDATE teams SET is_banned = true, banned_at = $2, banned_reason = $3
WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: UnbanTeam :one
UPDATE teams SET is_banned = false, banned_at = NULL, banned_reason = NULL
WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: SetTeamHidden :one
UPDATE teams SET is_hidden = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: UpdateTeamCaptain :one
UPDATE teams SET captain_id = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: GetAllTeams :many
SELECT id, name, invite_token, invite_token_expires_at, captain_id, bracket_id, is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden, avatar_url, created_at
FROM teams
WHERE deleted_at IS NULL
ORDER BY created_at ASC;

-- name: SetTeamBracket :one
UPDATE teams SET bracket_id = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: HardDeleteTeamsBefore :exec
DELETE FROM teams WHERE deleted_at IS NOT NULL AND deleted_at < $1;

-- name: UpdateTeamAdmin :one
UPDATE teams SET
    name = COALESCE(sqlc.narg('name'), name),
    captain_id = COALESCE(sqlc.narg('captain_id'), captain_id),
    bracket_id = COALESCE(sqlc.narg('bracket_id'), bracket_id),
    is_hidden = COALESCE(sqlc.narg('is_hidden'), is_hidden)
WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: UpdateTeamName :one
UPDATE teams SET name = $2 WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: UpdateInviteToken :one
UPDATE teams SET invite_token = $2, invite_token_expires_at = $3 WHERE id = $1 AND deleted_at IS NULL RETURNING id;

-- name: SearchTeams :many
SELECT id, name, invite_token, invite_token_expires_at, captain_id, bracket_id, is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden, avatar_url, created_at
FROM teams
WHERE deleted_at IS NULL
  AND is_hidden = false
  AND is_banned = false
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at ASC
LIMIT $1 OFFSET $2;

-- name: SearchTeamsAdmin :many
SELECT id, name, invite_token, invite_token_expires_at, captain_id, bracket_id, is_solo, is_auto_created, is_banned, banned_at, banned_reason, is_hidden, avatar_url, created_at
FROM teams
WHERE deleted_at IS NULL
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at ASC
LIMIT $1 OFFSET $2;

-- name: CountSearchTeamsAdmin :one
SELECT COUNT(*)::bigint FROM teams
WHERE deleted_at IS NULL
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%');

-- name: LockTeam :one
SELECT id FROM teams WHERE id = $1 AND deleted_at IS NULL FOR UPDATE;

-- name: CountSearchTeams :one
SELECT COUNT(*)::bigint
FROM teams
WHERE deleted_at IS NULL
  AND is_hidden = false
  AND is_banned = false
  AND (sqlc.narg('search')::text IS NULL OR name ILIKE '%' || sqlc.narg('search') || '%');

-- name: UpdateTeamAvatarURL :exec
UPDATE teams SET avatar_url = $2 WHERE id = $1 AND deleted_at IS NULL;

-- name: ClearTeamAvatarURL :exec
UPDATE teams SET avatar_url = NULL WHERE id = $1 AND deleted_at IS NULL;

-- name: ListAllTeamAvatarURLs :many
SELECT avatar_url FROM teams WHERE avatar_url IS NOT NULL AND deleted_at IS NULL;

