-- name: CreateHint :exec
INSERT INTO hints (id, challenge_id, title, content, cost, order_index)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetHintByID :one
SELECT id, challenge_id, title, content, cost, order_index
FROM hints
WHERE id = $1;

-- name: GetHintByIDForUpdate :one
SELECT id, challenge_id, title, content, cost, order_index
FROM hints
WHERE id = $1
FOR UPDATE;

-- name: GetHintsByChallengeID :many
SELECT id, challenge_id, title, content, cost, order_index
FROM hints
WHERE challenge_id = $1
ORDER BY order_index ASC;

-- name: GetHintsByChallengeIDs :many
SELECT id, challenge_id, title, content, cost, order_index
FROM hints
WHERE challenge_id = ANY($1::uuid[])
ORDER BY challenge_id, order_index ASC;

-- name: UpdateHint :exec
UPDATE hints SET title = $2, content = $3, cost = $4, order_index = $5 WHERE id = $1;

-- name: DeleteHint :exec
DELETE FROM hints WHERE id = $1;
