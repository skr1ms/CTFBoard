-- name: CreateTopic :one
INSERT INTO topics (id, name)
VALUES ($1, $2)
RETURNING id, name, created_at;

-- name: GetTopicByID :one
SELECT id, name, created_at FROM topics WHERE id = $1;

-- name: GetTopicByName :one
SELECT id, name, created_at FROM topics WHERE name = $1;

-- name: GetTopicsByIDs :many
SELECT id, name, created_at FROM topics WHERE id = ANY($1::uuid[]);

-- name: GetAllTopics :many
SELECT id, name, created_at FROM topics ORDER BY name;

-- name: UpdateTopic :exec
UPDATE topics SET name = $2 WHERE id = $1;

-- name: DeleteTopic :exec
DELETE FROM topics WHERE id = $1;

-- name: GetTopicsByChallengeID :many
SELECT t.id, t.name, t.created_at
FROM topics t
JOIN challenge_topics ct ON ct.topic_id = t.id
WHERE ct.challenge_id = $1
ORDER BY t.name;

-- name: GetTopicsByChallengeIDs :many
SELECT ct.challenge_id, t.id, t.name, t.created_at
FROM topics t
JOIN challenge_topics ct ON ct.topic_id = t.id
WHERE ct.challenge_id = ANY($1::uuid[])
ORDER BY ct.challenge_id, t.name;

-- name: DeleteChallengeTopics :exec
DELETE FROM challenge_topics WHERE challenge_id = $1;

-- name: InsertChallengeTopics :exec
INSERT INTO challenge_topics (challenge_id, topic_id)
SELECT sqlc.arg(challenge_id)::uuid, unnest(sqlc.arg(topic_ids)::uuid[])
ON CONFLICT (challenge_id, topic_id) DO NOTHING;
