-- name: CreateTag :exec
INSERT INTO tags (id, name, color)
VALUES ($1, $2, $3);

-- name: GetTagByID :one
SELECT id, name, color FROM tags WHERE id = $1;

-- name: GetTagByName :one
SELECT id, name, color FROM tags WHERE name = $1;

-- name: GetAllTags :many
SELECT id, name, color FROM tags ORDER BY name;

-- name: UpdateTag :exec
UPDATE tags SET name = $2, color = $3 WHERE id = $1;

-- name: DeleteTag :exec
DELETE FROM tags WHERE id = $1;

-- name: GetTagsByChallengeID :many
SELECT t.id, t.name, t.color
FROM tags t
JOIN challenge_tags ct ON ct.tag_id = t.id
WHERE ct.challenge_id = $1
ORDER BY t.name;

-- name: GetTagsByChallengeIDs :many
SELECT ct.challenge_id, t.id, t.name, t.color
FROM tags t
JOIN challenge_tags ct ON ct.tag_id = t.id
WHERE ct.challenge_id = ANY($1::uuid[])
ORDER BY ct.challenge_id, t.name;

-- name: DeleteChallengeTags :exec
DELETE FROM challenge_tags WHERE challenge_id = $1;

-- name: AddChallengeTag :exec
INSERT INTO challenge_tags (challenge_id, tag_id)
VALUES ($1, $2)
ON CONFLICT (challenge_id, tag_id) DO NOTHING;
