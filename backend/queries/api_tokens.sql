-- name: CreateAPIToken :exec
INSERT INTO api_tokens (id, user_id, token_hash, description, expires_at, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetAPITokensByUserID :many
SELECT id, user_id, token_hash, description, expires_at, last_used_at, created_at
FROM api_tokens
WHERE user_id = $1
ORDER BY created_at DESC;

-- name: GetAPITokenByHash :one
SELECT id, user_id, token_hash, description, expires_at, last_used_at, created_at
FROM api_tokens
WHERE token_hash = $1;

-- name: DeleteAPIToken :exec
DELETE FROM api_tokens WHERE id = $1 AND user_id = $2;

-- name: UpdateAPITokenLastUsed :exec
UPDATE api_tokens SET last_used_at = $2 WHERE id = $1;
