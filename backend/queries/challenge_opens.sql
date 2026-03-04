-- name: CreateChallengeOpen :exec
INSERT INTO challenge_opens (id, user_id, challenge_id, ip)
VALUES ($1, $2, $3, $4);

-- name: GetChallengeOpensByChallenge :many
SELECT id, user_id, challenge_id, ip, opened_at
FROM challenge_opens
WHERE challenge_id = $1
ORDER BY opened_at DESC
LIMIT $2 OFFSET $3;

-- name: CountChallengeOpensByChallenge :one
SELECT COUNT(*)::int FROM challenge_opens WHERE challenge_id = $1;
