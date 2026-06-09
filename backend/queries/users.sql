-- name: LockUser :one
SELECT id FROM users WHERE id = $1 FOR UPDATE;

-- name: CreateUser :exec
INSERT INTO users (id, username, email, password_hash, role, is_verified, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetUserByID :one
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, was_in_banned_team, avatar_url, created_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, was_in_banned_team, avatar_url, created_at
FROM users
WHERE email = $1;

-- name: GetUserByUsername :one
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, was_in_banned_team, avatar_url, created_at
FROM users
WHERE username = $1;

-- name: UpdateUserTeamID :one
UPDATE users SET team_id = $2 WHERE id = $1 RETURNING id;

-- name: UpdateUserTeamIDBatch :exec
UPDATE users SET team_id = $1 WHERE id = ANY($2::uuid[]);

-- name: ListUserIDsWithTeamIDNull :many
SELECT id FROM users WHERE id = ANY($1::uuid[]) AND team_id IS NULL;

-- name: ListUserIDsWithTeamIDNullAndNotBanned :many
SELECT id FROM users WHERE id = ANY($1::uuid[]) AND team_id IS NULL AND is_banned = false;

-- name: UpdateUserVerified :one
UPDATE users SET is_verified = $2, verified_at = $3 WHERE id = $1 RETURNING id;

-- name: ListUsersByTeamID :many
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, was_in_banned_team, avatar_url, created_at
FROM users
WHERE team_id = $1;

-- name: ListUsersByTeamIDs :many
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, was_in_banned_team, avatar_url, created_at
FROM users
WHERE team_id = ANY($1::uuid[])
ORDER BY team_id, created_at;

-- name: GetAllUsers :many
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, was_in_banned_team, avatar_url, created_at
FROM users
ORDER BY created_at ASC;

-- name: UpdatePassword :execrows
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: SearchUsers :many
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, was_in_banned_team, avatar_url, created_at
FROM users
WHERE (sqlc.narg('search')::text IS NULL OR username ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at ASC
LIMIT $1 OFFSET $2;

-- name: CountSearchUsers :one
SELECT COUNT(*)::bigint
FROM users
WHERE (sqlc.narg('search')::text IS NULL OR username ILIKE '%' || sqlc.narg('search') || '%');

-- name: SearchUsersAdmin :many
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, was_in_banned_team, avatar_url, created_at
FROM users
WHERE (
    sqlc.narg('search')::text IS NULL
    OR username ILIKE '%' || sqlc.narg('search') || '%'
    OR email ILIKE '%' || sqlc.narg('search') || '%'
  )
  AND (
    sqlc.arg('ban_status')::text = 'all'
    OR (sqlc.arg('ban_status')::text = 'direct' AND is_banned = true)
    OR (sqlc.arg('ban_status')::text = 'team_inherited' AND is_banned = false AND was_in_banned_team = true AND role <> 'admin')
    OR (sqlc.arg('ban_status')::text = 'blocked' AND (is_banned = true OR (was_in_banned_team = true AND role <> 'admin')))
    OR (sqlc.arg('ban_status')::text = 'not_banned' AND is_banned = false AND NOT (was_in_banned_team = true AND role <> 'admin'))
  )
ORDER BY created_at ASC
LIMIT $1 OFFSET $2;

-- name: CountSearchUsersAdmin :one
SELECT COUNT(*)::bigint
FROM users
WHERE (
    sqlc.narg('search')::text IS NULL
    OR username ILIKE '%' || sqlc.narg('search') || '%'
    OR email ILIKE '%' || sqlc.narg('search') || '%'
  )
  AND (
    sqlc.arg('ban_status')::text = 'all'
    OR (sqlc.arg('ban_status')::text = 'direct' AND is_banned = true)
    OR (sqlc.arg('ban_status')::text = 'team_inherited' AND is_banned = false AND was_in_banned_team = true AND role <> 'admin')
    OR (sqlc.arg('ban_status')::text = 'blocked' AND (is_banned = true OR (was_in_banned_team = true AND role <> 'admin')))
    OR (sqlc.arg('ban_status')::text = 'not_banned' AND is_banned = false AND NOT (was_in_banned_team = true AND role <> 'admin'))
  );

-- name: UpdateUserAdmin :execrows
UPDATE users SET
    username = COALESCE(sqlc.narg('username'), username),
    email = COALESCE(sqlc.narg('email'), email),
    role = COALESCE(sqlc.narg('role'), role),
    is_verified = COALESCE(sqlc.narg('is_verified'), is_verified),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash)
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: UpdateUserProfile :execrows
UPDATE users SET
    username = COALESCE(sqlc.narg('username'), username),
    email = COALESCE(sqlc.narg('email'), email),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash)
WHERE id = $1;

-- name: SearchUsersByIP :many
SELECT DISTINCT u.id, u.team_id, u.username, u.email, u.password_hash, u.role, u.is_verified, u.verified_at, u.is_banned, u.banned_at, u.banned_reason, u.was_in_banned_team, u.avatar_url, u.created_at
FROM users u
INNER JOIN tracking t ON t.user_id = u.id
WHERE t.ip = $1
ORDER BY u.created_at ASC
LIMIT $2 OFFSET $3;

-- name: CountSearchUsersByIP :one
SELECT COUNT(DISTINCT u.id)::bigint
FROM users u
INNER JOIN tracking t ON t.user_id = u.id
WHERE t.ip = $1;

-- name: SearchUsersAdminByIP :many
SELECT DISTINCT u.id, u.team_id, u.username, u.email, u.password_hash, u.role, u.is_verified, u.verified_at, u.is_banned, u.banned_at, u.banned_reason, u.was_in_banned_team, u.avatar_url, u.created_at
FROM users u
INNER JOIN tracking t ON t.user_id = u.id
WHERE t.ip = $1
  AND (
    sqlc.arg('ban_status')::text = 'all'
    OR (sqlc.arg('ban_status')::text = 'direct' AND u.is_banned = true)
    OR (sqlc.arg('ban_status')::text = 'team_inherited' AND u.is_banned = false AND u.was_in_banned_team = true AND u.role <> 'admin')
    OR (sqlc.arg('ban_status')::text = 'blocked' AND (u.is_banned = true OR (u.was_in_banned_team = true AND u.role <> 'admin')))
    OR (sqlc.arg('ban_status')::text = 'not_banned' AND u.is_banned = false AND NOT (u.was_in_banned_team = true AND u.role <> 'admin'))
  )
ORDER BY u.created_at ASC
LIMIT $2 OFFSET $3;

-- name: CountSearchUsersAdminByIP :one
SELECT COUNT(DISTINCT u.id)::bigint
FROM users u
INNER JOIN tracking t ON t.user_id = u.id
WHERE t.ip = $1
  AND (
    sqlc.arg('ban_status')::text = 'all'
    OR (sqlc.arg('ban_status')::text = 'direct' AND u.is_banned = true)
    OR (sqlc.arg('ban_status')::text = 'team_inherited' AND u.is_banned = false AND u.was_in_banned_team = true AND u.role <> 'admin')
    OR (sqlc.arg('ban_status')::text = 'blocked' AND (u.is_banned = true OR (u.was_in_banned_team = true AND u.role <> 'admin')))
    OR (sqlc.arg('ban_status')::text = 'not_banned' AND u.is_banned = false AND NOT (u.was_in_banned_team = true AND u.role <> 'admin'))
  );

-- name: CountActiveUsers :one
SELECT COUNT(*)::bigint
FROM users
WHERE role = 'user' AND is_banned = false;

-- name: BanUser :one
UPDATE users SET is_banned = TRUE, banned_at = $2, banned_reason = $3 WHERE id = $1 RETURNING id;

-- name: UnbanUser :one
UPDATE users SET is_banned = FALSE, banned_at = NULL, banned_reason = NULL WHERE id = $1 RETURNING id;

-- name: SetWasInBannedTeamByIDs :exec
UPDATE users SET was_in_banned_team = $1 WHERE id = ANY($2::uuid[]);

-- name: UpdateUserAvatarURL :exec
UPDATE users SET avatar_url = $2 WHERE id = $1;

-- name: ClearUserAvatarURL :exec
UPDATE users SET avatar_url = NULL WHERE id = $1;

-- name: ListAllUserAvatarURLs :many
SELECT avatar_url FROM users WHERE avatar_url IS NOT NULL;
