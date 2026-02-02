-- name: CreateHint :exec
INSERT INTO hints (id, challenge_id, content, cost, order_index)
VALUES ($1, $2, $3, $4, $5);

-- name: GetHintByID :one
SELECT id, challenge_id, content, cost, order_index
FROM hints
WHERE id = $1;

-- name: GetHintsByChallengeID :many
SELECT id, challenge_id, content, cost, order_index
FROM hints
WHERE challenge_id = $1
ORDER BY order_index ASC;

-- name: UpdateHint :exec
UPDATE hints SET content = $2, cost = $3, order_index = $4 WHERE id = $1;

-- name: DeleteHint :exec
DELETE FROM hints WHERE id = $1;
