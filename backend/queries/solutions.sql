-- name: GetSolutionByChallengeID :one
SELECT * FROM solutions WHERE challenge_id = $1;

-- name: UpsertSolution :one
INSERT INTO solutions (challenge_id, content)
VALUES ($1, $2)
ON CONFLICT (challenge_id) DO UPDATE SET content = EXCLUDED.content
RETURNING *;

-- name: DeleteSolution :exec
DELETE FROM solutions WHERE challenge_id = $1;

-- name: GetSolutionsByTeamID :many
SELECT
    s.challenge_id,
    s.content,
    c.title    AS challenge_title,
    c.category AS challenge_category
FROM solutions s
JOIN challenges c ON c.id = s.challenge_id
WHERE EXISTS (
    SELECT 1 FROM solves sv
    WHERE sv.challenge_id = s.challenge_id
      AND sv.team_id = $1
)
ORDER BY c.category, c.title;

-- name: GetWriteupFilesByIDs :many
SELECT * FROM files
WHERE type = 'writeup'
  AND challenge_id = ANY($1::uuid[])
ORDER BY challenge_id, created_at;
