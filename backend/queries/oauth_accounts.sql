-- name: CreateOAuthAccount :exec
INSERT INTO oauth_accounts (id, user_id, provider, provider_user_id, access_token, refresh_token, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetOAuthAccount :one
SELECT * FROM oauth_accounts
WHERE provider = $1 AND provider_user_id = $2
LIMIT 1;

-- name: GetOAuthAccountsByUserID :many
SELECT * FROM oauth_accounts WHERE user_id = $1;

-- name: UpsertOAuthAccount :exec
INSERT INTO oauth_accounts (id, user_id, provider, provider_user_id, access_token, refresh_token, expires_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT (provider, provider_user_id)
DO UPDATE SET access_token = EXCLUDED.access_token,
             refresh_token = EXCLUDED.refresh_token,
             expires_at    = EXCLUDED.expires_at;
