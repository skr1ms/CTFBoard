-- name: CreateUser :exec
INSERT INTO users (id, username, email, password_hash, role, is_verified, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: CreateUserReturningID :one
INSERT INTO users (username, email, password_hash, role, is_verified, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: GetUserByID :one
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, created_at
FROM users
WHERE id = $1;

-- name: GetUserByEmail :one
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, created_at
FROM users
WHERE email = $1;

-- name: GetUserByUsername :one
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, created_at
FROM users
WHERE username = $1;

-- name: UpdateUserTeamID :one
UPDATE users SET team_id = $2 WHERE id = $1 RETURNING id;

-- name: UpdateUserVerified :exec
UPDATE users SET is_verified = $2, verified_at = $3 WHERE id = $1;

-- name: ListUsersByTeamID :many
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, created_at
FROM users
WHERE team_id = $1;

-- name: GetAllUsers :many
SELECT id, team_id, username, email, password_hash, role, is_verified, verified_at, created_at
FROM users
ORDER BY created_at ASC;

-- name: UpdatePassword :exec
UPDATE users SET password_hash = $2 WHERE id = $1;
