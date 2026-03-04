-- name: LockUser :one
SELECT id FROM users WHERE id = $1 FOR UPDATE;

-- name: CreateUser :exec
INSERT INTO users (id, username, email, password_hash, role, is_verified, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: CreateUserReturningID :one
INSERT INTO users (username, email, password_hash, role, is_verified, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: GetUserByID :one
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, created_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, created_at
FROM users
WHERE email = $1;

-- name: GetUserByUsername :one
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, created_at
FROM users
WHERE username = $1;

-- name: UpdateUserTeamID :one
UPDATE users SET team_id = $2 WHERE id = $1 RETURNING id;

-- name: UpdateUserVerified :one
UPDATE users SET is_verified = $2, verified_at = $3 WHERE id = $1 RETURNING id;

-- name: ListUsersByTeamID :many
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, created_at
FROM users
WHERE team_id = $1;

-- name: GetAllUsers :many
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, created_at
FROM users
ORDER BY created_at ASC;

-- name: UpdatePassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;

-- name: SearchUsers :many
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, is_banned, banned_at, banned_reason, created_at
FROM users
WHERE (sqlc.narg('search')::text IS NULL OR username ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY created_at ASC
LIMIT $1 OFFSET $2;

-- name: CountSearchUsers :one
SELECT COUNT(*)
FROM users
WHERE (sqlc.narg('search')::text IS NULL OR username ILIKE '%' || sqlc.narg('search') || '%');

-- name: UpdateUserAdmin :exec
UPDATE users SET
    username = COALESCE(sqlc.narg('username'), username),
    email = COALESCE(sqlc.narg('email'), email),
    role = COALESCE(sqlc.narg('role'), role),
    is_verified = COALESCE(sqlc.narg('is_verified'), is_verified),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash)
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users WHERE id = $1;

-- name: UpdateUserProfile :exec
UPDATE users SET
    username = COALESCE(sqlc.narg('username'), username),
    email = COALESCE(sqlc.narg('email'), email),
    password_hash = COALESCE(sqlc.narg('password_hash'), password_hash)
WHERE id = $1;

-- name: SearchUsersByIP :many
SELECT DISTINCT u.id, u.team_id, u.username, u.email, u.password_hash, u.role, u.is_verified, u.verified_at, u.is_banned, u.banned_at, u.banned_reason, u.created_at
FROM users u
INNER JOIN tracking t ON t.user_id = u.id
WHERE t.ip = $1
ORDER BY u.created_at ASC
LIMIT $2 OFFSET $3;

-- name: CountSearchUsersByIP :one
SELECT COUNT(DISTINCT u.id)
FROM users u
INNER JOIN tracking t ON t.user_id = u.id
WHERE t.ip = $1;

-- name: BanUser :one
UPDATE users SET is_banned = TRUE, banned_at = $2, banned_reason = $3 WHERE id = $1 RETURNING id;

-- name: UnbanUser :one
UPDATE users SET is_banned = FALSE, banned_at = NULL, banned_reason = NULL WHERE id = $1 RETURNING id;
