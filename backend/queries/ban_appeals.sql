-- name: CreateBanAppeal :one
INSERT INTO ban_appeals (user_id, message)
VALUES ($1, $2)
RETURNING id, user_id, message, created_at, reviewed_at, admin_response, decision;

-- name: GetBanAppealByID :one
SELECT id, user_id, message, created_at, reviewed_at, admin_response, decision
FROM ban_appeals
WHERE id = $1;

-- name: GetBanAppealsByUserID :many
SELECT id, user_id, message, created_at, reviewed_at, admin_response, decision
FROM ban_appeals
WHERE user_id = $1
ORDER BY created_at DESC, id DESC;

-- name: GetLatestBanAppealByUserID :one
SELECT id, user_id, message, created_at, reviewed_at, admin_response, decision
FROM ban_appeals
WHERE user_id = $1
ORDER BY created_at DESC, id DESC
LIMIT 1;

-- name: ListBanAppeals :many
SELECT id, user_id, message, created_at, reviewed_at, admin_response, decision
FROM ban_appeals
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: ListBanAppealsByDecision :many
SELECT id, user_id, message, created_at, reviewed_at, admin_response, decision
FROM ban_appeals
WHERE decision = $1
ORDER BY created_at DESC, id DESC
LIMIT $2 OFFSET $3;

-- name: CountBanAppeals :one
SELECT COUNT(*) FROM ban_appeals;

-- name: CountBanAppealsByDecision :one
SELECT COUNT(*) FROM ban_appeals WHERE decision = $1;

-- name: UpdateBanAppeal :execrows
UPDATE ban_appeals
SET decision = $2, admin_response = $3, reviewed_at = $4
WHERE id = $1;
