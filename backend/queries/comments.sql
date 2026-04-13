-- name: CreateComment :one
INSERT INTO comments (id, user_id, challenge_id, content, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, user_id, challenge_id, content, created_at, updated_at;

-- name: GetCommentByID :one
SELECT c.id, c.user_id, c.challenge_id, c.content, c.created_at, c.updated_at, u.username
FROM comments c
JOIN users u ON u.id = c.user_id
WHERE c.id = $1;

-- name: GetCommentsByChallengeID :many
SELECT c.id, c.user_id, c.challenge_id, c.content, c.created_at, c.updated_at, u.username
FROM comments c
JOIN users u ON u.id = c.user_id
WHERE c.challenge_id = $1 ORDER BY c.created_at ASC;

-- name: GetAllComments :many
SELECT c.id, c.user_id, c.challenge_id, c.content, c.created_at, c.updated_at, u.username
FROM comments c
JOIN users u ON u.id = c.user_id
ORDER BY c.created_at ASC;

-- name: UpdateComment :exec
UPDATE comments SET content = $2, updated_at = $3 WHERE id = $1;

-- name: DeleteComment :exec
DELETE FROM comments WHERE id = $1;
