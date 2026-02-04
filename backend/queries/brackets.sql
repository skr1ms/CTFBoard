-- name: CreateBracket :one
INSERT INTO brackets (id, name, description, is_default, created_at)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, name, description, is_default, created_at;

-- name: GetBracketByID :one
SELECT id, name, description, is_default, created_at
FROM brackets WHERE id = $1;

-- name: GetBracketByName :one
SELECT id, name, description, is_default, created_at
FROM brackets WHERE name = $1;

-- name: GetAllBrackets :many
SELECT id, name, description, is_default, created_at
FROM brackets ORDER BY name ASC;

-- name: UpdateBracket :exec
UPDATE brackets SET name = $2, description = $3, is_default = $4
WHERE id = $1;

-- name: DeleteBracket :exec
DELETE FROM brackets WHERE id = $1;
