-- name: UpsertRating :one
INSERT INTO ratings (id, challenge_id, user_id, team_id, value, review, updated_at)
VALUES (uuid_generate_v4(), $1, $2, $3, $4, $5, $6)
ON CONFLICT (team_id, challenge_id) DO UPDATE SET
    user_id = EXCLUDED.user_id,
    value = EXCLUDED.value,
    review = EXCLUDED.review,
    updated_at = EXCLUDED.updated_at,
    banned_team_id = NULL
RETURNING id, challenge_id, user_id, team_id, banned_team_id, value, review, created_at, updated_at;

-- name: GetRatingsByChallengeID :many
SELECT r.id, r.challenge_id, r.user_id, r.team_id, r.banned_team_id, r.value, r.review, r.created_at, r.updated_at
FROM ratings AS r
JOIN teams AS t ON t.id = r.team_id
WHERE r.challenge_id = $1
  AND r.banned_team_id IS NULL
  AND t.deleted_at IS NULL
  AND t.is_banned = FALSE
  AND t.is_hidden = FALSE
ORDER BY r.created_at ASC;

-- name: GetRatingByTeamAndChallenge :one
SELECT id, challenge_id, user_id, team_id, banned_team_id, value, review, created_at, updated_at
FROM ratings
WHERE team_id = $1 AND challenge_id = $2 AND banned_team_id IS NULL;

-- name: GetAllRatings :many
SELECT id, challenge_id, user_id, team_id, banned_team_id, value, review, created_at, updated_at
FROM ratings
ORDER BY created_at ASC;

-- name: DeleteRatingsByTeamID :exec
DELETE FROM ratings WHERE team_id = $1;

-- name: SoftBanRatingsByTeamID :exec
UPDATE ratings
SET banned_team_id = $1,
    updated_at = CURRENT_TIMESTAMP
WHERE team_id = $1 AND banned_team_id IS NULL;

-- name: RestoreRatingsByBannedTeamID :exec
UPDATE ratings
SET banned_team_id = NULL,
    updated_at = CURRENT_TIMESTAMP
WHERE banned_team_id = $1;
