-- name: UpsertRating :one
INSERT INTO ratings (id, challenge_id, user_id, team_id, value, review, updated_at)
VALUES (uuid_generate_v4(), $1, $2, $3, $4, $5, $6)
ON CONFLICT (team_id, challenge_id) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    value = EXCLUDED.value,
    review = EXCLUDED.review,
    updated_at = EXCLUDED.updated_at
RETURNING id, challenge_id, user_id, team_id, value, review, created_at, updated_at;

-- name: GetRatingsByChallengeID :many
SELECT id, challenge_id, user_id, team_id, value, review, created_at, updated_at
FROM ratings
WHERE challenge_id = $1
ORDER BY created_at ASC;

-- name: GetRatingByTeamAndChallenge :one
SELECT id, challenge_id, user_id, team_id, value, review, created_at, updated_at
FROM ratings
WHERE team_id = $1 AND challenge_id = $2;

-- name: GetAllRatings :many
SELECT id, challenge_id, user_id, team_id, value, review, created_at, updated_at
FROM ratings
ORDER BY created_at ASC;

-- name: DeleteRatingsByTeamID :exec
DELETE FROM ratings WHERE team_id = $1;
