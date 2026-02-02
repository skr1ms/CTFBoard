-- name: CreateVerificationToken :exec
INSERT INTO verification_tokens (id, user_id, token, type, expires_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetVerificationTokenByToken :one
SELECT id, user_id, token, type, expires_at, used_at, created_at
FROM verification_tokens
WHERE token = $1;

-- name: MarkVerificationTokenUsed :exec
UPDATE verification_tokens SET used_at = NOW() WHERE id = $1;

-- name: DeleteExpiredVerificationTokens :exec
DELETE FROM verification_tokens WHERE expires_at < $1;

-- name: DeleteVerificationTokensByUserAndType :exec
DELETE FROM verification_tokens WHERE user_id = $1 AND type = $2;
