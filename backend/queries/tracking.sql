-- name: CreateTracking :exec
INSERT INTO tracking (id, user_id, ip, user_agent)
VALUES ($1, $2, $3, $4);

-- name: GetTrackingByUser :many
SELECT id, user_id, ip, user_agent, tracked_at
FROM tracking
WHERE user_id = $1
ORDER BY tracked_at DESC
LIMIT $2 OFFSET $3;

-- name: CountTrackingByUser :one
SELECT COUNT(*)::int FROM tracking WHERE user_id = $1;
