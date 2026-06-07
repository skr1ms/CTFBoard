-- name: GetSolutionByChallengeID :one
SELECT id, challenge_id, content, state FROM solutions WHERE challenge_id = $1;

-- name: GetAllSolutions :many
SELECT id, challenge_id, content, state FROM solutions ORDER BY challenge_id;

-- name: UpsertSolution :one
INSERT INTO solutions (challenge_id, content, state)
VALUES ($1, $2, $3)
ON CONFLICT (challenge_id) DO UPDATE SET content = EXCLUDED.content, state = EXCLUDED.state
RETURNING id, challenge_id, content, state;

-- name: DeleteSolution :exec
DELETE FROM solutions WHERE challenge_id = $1;

-- name: GetCandidateSolutions :many
SELECT
    s.challenge_id,
    s.content,
    s.state,
    c.title    AS challenge_title,
    c.category AS challenge_category
FROM solutions s
JOIN challenges c ON c.id = s.challenge_id
WHERE c.state IN ('visible', 'locked')
ORDER BY c.category, c.title;
