-- name: CreateComment :one
INSERT INTO comments (id, user_id, challenge_id, content, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, challenge_id, content, created_at, updated_at;

-- name: GetCommentByID :one
SELECT id, user_id, challenge_id, content, created_at, updated_at
FROM comments WHERE id = $1;

-- name: GetCommentsByChallengeID :many
SELECT id, user_id, challenge_id, content, created_at, updated_at
FROM comments WHERE challenge_id = $1 ORDER BY created_at ASC;

-- name: UpdateComment :exec
UPDATE comments SET content = $2, updated_at = $3 WHERE id = $1;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;
