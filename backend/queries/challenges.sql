-- name: CreateChallenge :exec
INSERT INTO challenges (id, title, description, category, points, initial_value, min_value, decay, solve_count, flag_hash, is_hidden, is_regex, is_case_insensitive, flag_regex, flag_format_regex)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15);

-- name: GetChallengeByID :one
SELECT id, title, description, category, points, initial_value, min_value, decay, solve_count, flag_hash, is_hidden, is_regex, is_case_insensitive, flag_regex, flag_format_regex
FROM challenges
WHERE id = $1;

-- name: GetChallengeByIDForUpdate :one
SELECT id, title, description, category, points, initial_value, min_value, decay, solve_count, flag_hash, is_hidden, is_regex, is_case_insensitive, flag_regex, flag_format_regex
FROM challenges
WHERE id = $1
FOR UPDATE;

-- name: GetChallengesByIDs :many
SELECT id, title, description, category, points, initial_value, min_value, decay, solve_count, flag_hash, is_hidden, is_regex, is_case_insensitive, flag_regex, flag_format_regex
FROM challenges
WHERE id = ANY($1::uuid[]);

-- name: GetChallenges :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.is_hidden, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex, 0::int as solved
FROM challenges c
WHERE c.is_hidden = false;

-- name: GetChallengesByTag :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.is_hidden, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex, 0::int as solved
FROM challenges c
JOIN challenge_tags ct ON ct.challenge_id = c.id AND ct.tag_id = $1
WHERE c.is_hidden = false;

-- name: GetChallengesForTeam :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.is_hidden, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex,
    (CASE WHEN s.id IS NOT NULL THEN 1 ELSE 0 END)::int AS solved
FROM challenges c
LEFT JOIN solves s ON s.challenge_id = c.id AND s.team_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
WHERE c.is_hidden = false;

-- name: GetChallengesForTeamByTag :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.is_hidden, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex,
    (CASE WHEN s.id IS NOT NULL THEN 1 ELSE 0 END)::int AS solved
FROM challenges c
JOIN challenge_tags ct ON ct.challenge_id = c.id AND ct.tag_id = $1
LEFT JOIN solves s ON s.challenge_id = c.id AND s.team_id = $2 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
WHERE c.is_hidden = false;

-- name: UpdateChallenge :exec
UPDATE challenges SET
    title = $2, description = $3, category = $4, points = $5, initial_value = $6, min_value = $7,
    decay = $8, flag_hash = $9, is_hidden = $10, is_regex = $11, is_case_insensitive = $12, flag_regex = $13, flag_format_regex = $14
WHERE id = $1;

-- name: DeleteChallenge :one
DELETE FROM challenges WHERE id = $1 RETURNING id;

-- name: IncrementChallengeSolveCount :one
UPDATE challenges SET solve_count = solve_count + 1 WHERE id = $1 RETURNING solve_count;

-- name: DecrementChallengeSolveCount :one
UPDATE challenges SET solve_count = GREATEST(solve_count - 1, 0) WHERE id = $1 RETURNING solve_count;

-- name: BatchDecrementChallengeSolveCount :exec
UPDATE challenges SET solve_count = GREATEST(solve_count - 1, 0) WHERE id = ANY($1::uuid[]);

-- name: BatchIncrementChallengeSolveCount :exec
UPDATE challenges SET solve_count = solve_count + 1 WHERE id = ANY($1::uuid[]);

-- name: UpdateChallengePoints :one
UPDATE challenges SET points = $2 WHERE id = $1 RETURNING id;

-- name: BatchUpdateChallengePoints :exec
UPDATE challenges AS c SET points = v.points
FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS points) AS v
WHERE c.id = v.id;

-- name: GetChallengeFlags :one
SELECT flag_hash, is_regex, is_case_insensitive, flag_regex, flag_format_regex
FROM challenges
WHERE id = $1;

-- name: GetMissingChallengesByTeamID :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.is_hidden, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex
FROM challenges c
WHERE c.is_hidden = false
  AND NOT EXISTS (
    SELECT 1 FROM solves s
    WHERE s.challenge_id = c.id AND s.team_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
  )
ORDER BY c.category, c.points DESC;

-- name: GetMissingChallengesByUserID :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.is_hidden, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex
FROM challenges c
WHERE c.is_hidden = false
  AND NOT EXISTS (
    SELECT 1 FROM solves s
    WHERE s.challenge_id = c.id AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND s.team_id = (SELECT u.team_id FROM users u WHERE u.id = $1 AND u.team_id IS NOT NULL)
  )
ORDER BY c.category, c.points DESC;

-- name: GetChallengeRequirements :many
SELECT c.id, c.title, c.category
FROM challenge_requirements cr
JOIN challenges c ON c.id = cr.required_challenge_id
WHERE cr.challenge_id = $1
  AND c.is_hidden = false
ORDER BY c.title;

-- name: DeleteChallengeRequirements :exec
DELETE FROM challenge_requirements WHERE challenge_id = $1;

-- name: GetAllChallengeRequirements :many
SELECT challenge_id, required_challenge_id FROM challenge_requirements ORDER BY challenge_id, required_challenge_id;
