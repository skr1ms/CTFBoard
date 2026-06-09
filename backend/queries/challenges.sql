-- name: CreateChallenge :exec
INSERT INTO challenges (id, title, description, category, points, initial_value, min_value, decay, solve_count, flag_hash, attribution, connection_info, max_attempts, max_attempts_window, position, next_challenge_id, state, is_regex, is_case_insensitive, flag_regex, flag_format_regex, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22, $23);

-- name: GetChallengeByID :one
SELECT id, title, description, category, points, initial_value, min_value, decay, solve_count, flag_hash, attribution, connection_info, max_attempts, max_attempts_window, position, next_challenge_id, state, is_regex, is_case_insensitive, flag_regex, flag_format_regex, created_at, updated_at
FROM challenges
WHERE id = $1;

-- name: GetChallengeByIDForUpdate :one
SELECT id, title, description, category, points, initial_value, min_value, decay, solve_count, flag_hash, attribution, connection_info, max_attempts, max_attempts_window, position, next_challenge_id, state, is_regex, is_case_insensitive, flag_regex, flag_format_regex, created_at, updated_at
FROM challenges
WHERE id = $1
FOR UPDATE;

-- name: GetChallengesByIDs :many
SELECT id, title, description, category, points, initial_value, min_value, decay, solve_count, flag_hash, attribution, connection_info, max_attempts, max_attempts_window, position, next_challenge_id, state, is_regex, is_case_insensitive, flag_regex, flag_format_regex, created_at, updated_at
FROM challenges
WHERE id = ANY($1::uuid[]);

-- name: GetChallenges :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.attribution, c.connection_info, c.max_attempts, c.max_attempts_window, c.position, c.next_challenge_id, c.state, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex, c.created_at, c.updated_at, 0::int as solved
FROM challenges c
WHERE c.state IN ('visible', 'locked')
ORDER BY c.position ASC, c.id ASC;

-- name: GetChallengesAll :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.attribution, c.connection_info, c.max_attempts, c.max_attempts_window, c.position, c.next_challenge_id, c.state, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex, c.created_at, c.updated_at, 0::int as solved
FROM challenges c
ORDER BY c.position ASC, c.id ASC;

-- name: GetChallengesByTag :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.attribution, c.connection_info, c.max_attempts, c.max_attempts_window, c.position, c.next_challenge_id, c.state, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex, c.created_at, c.updated_at, 0::int as solved
FROM challenges c
JOIN challenge_tags ct ON ct.challenge_id = c.id AND ct.tag_id = $1
WHERE c.state IN ('visible', 'locked')
ORDER BY c.position ASC, c.id ASC;

-- name: GetChallengesForTeam :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.attribution, c.connection_info, c.max_attempts, c.max_attempts_window, c.position, c.next_challenge_id, c.state, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex, c.created_at, c.updated_at,
    (CASE WHEN s.id IS NOT NULL THEN 1 ELSE 0 END)::int AS solved
FROM challenges c
LEFT JOIN solves s ON s.challenge_id = c.id AND s.team_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
WHERE c.state IN ('visible', 'locked')
ORDER BY c.position ASC, c.id ASC;

-- name: GetChallengesForTeamByTag :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.attribution, c.connection_info, c.max_attempts, c.max_attempts_window, c.position, c.next_challenge_id, c.state, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex, c.created_at, c.updated_at,
    (CASE WHEN s.id IS NOT NULL THEN 1 ELSE 0 END)::int AS solved
FROM challenges c
JOIN challenge_tags ct ON ct.challenge_id = c.id AND ct.tag_id = $1
LEFT JOIN solves s ON s.challenge_id = c.id AND s.team_id = $2 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
WHERE c.state IN ('visible', 'locked')
ORDER BY c.position ASC, c.id ASC;

-- name: UpdateChallenge :exec
UPDATE challenges SET
    title = $2, description = $3, category = $4, points = $5, initial_value = $6, min_value = $7,
    decay = $8, flag_hash = $9, attribution = $10, connection_info = $11, max_attempts = $12, max_attempts_window = $13, position = $14, next_challenge_id = $15, state = $16, is_regex = $17, is_case_insensitive = $18, flag_regex = $19, flag_format_regex = $20, updated_at = $21
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

-- name: RecalculateChallengeSolveCounts :exec
UPDATE challenges AS c
SET solve_count = COALESCE(v.cnt, 0)
FROM (
    SELECT s.challenge_id, COUNT(*)::int AS cnt
    FROM solves s
    JOIN teams t ON t.id = s.team_id AND t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
    WHERE s.challenge_id = ANY($1::uuid[])
      AND s.banned_team_id IS NULL
      AND s.banned_user_id IS NULL
    GROUP BY s.challenge_id
) v
WHERE c.id = v.challenge_id;

-- name: ZeroChallengeSolveCountsIfNoSolves :exec
UPDATE challenges SET solve_count = 0
WHERE id = ANY($1::uuid[])
  AND id NOT IN (
    SELECT DISTINCT s.challenge_id FROM solves s
    JOIN teams t ON t.id = s.team_id AND t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
    WHERE s.challenge_id = ANY($1::uuid[])
      AND s.banned_team_id IS NULL
      AND s.banned_user_id IS NULL
  );

-- name: GetAllDynamicChallengeIDs :many
SELECT id FROM challenges WHERE initial_value > 0 AND decay > 0;

-- name: GetChallengeFlags :one
SELECT flag_hash, is_regex, is_case_insensitive, flag_regex, flag_format_regex
FROM challenges
WHERE id = $1;

-- name: GetMissingChallengesByTeamID :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.attribution, c.connection_info, c.max_attempts, c.max_attempts_window, c.position, c.next_challenge_id, c.state, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex, c.created_at, c.updated_at
FROM challenges c
WHERE c.state IN ('visible', 'locked')
  AND NOT EXISTS (
    SELECT 1 FROM solves s
    WHERE s.challenge_id = c.id AND s.team_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
  )
ORDER BY c.position ASC, c.id ASC;

-- name: GetMissingChallengesByUserID :many
SELECT c.id, c.title, c.description, c.category, c.points, c.initial_value, c.min_value, c.decay, c.solve_count, c.flag_hash, c.attribution, c.connection_info, c.max_attempts, c.max_attempts_window, c.position, c.next_challenge_id, c.state, c.is_regex, c.is_case_insensitive, c.flag_regex, c.flag_format_regex, c.created_at, c.updated_at
FROM challenges c
WHERE c.state IN ('visible', 'locked')
  AND NOT EXISTS (
    SELECT 1 FROM solves s
    WHERE s.challenge_id = c.id AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND s.team_id = (SELECT u.team_id FROM users u WHERE u.id = $1 AND u.team_id IS NOT NULL)
  )
ORDER BY c.position ASC, c.id ASC;

-- name: GetChallengeRequirements :many
SELECT c.id, c.title, c.category
FROM challenge_requirements cr
JOIN challenges c ON c.id = cr.required_challenge_id
WHERE cr.challenge_id = $1
  AND c.state IN ('visible', 'locked')
ORDER BY c.title;

-- name: GetChallengeRequirementsForEnforcement :many
SELECT c.id, c.title, c.category
FROM challenge_requirements cr
JOIN challenges c ON c.id = cr.required_challenge_id
WHERE cr.challenge_id = $1
ORDER BY c.title;

-- name: DeleteChallengeRequirements :exec
DELETE FROM challenge_requirements WHERE challenge_id = $1;

-- name: InsertChallengeRequirements :exec
INSERT INTO challenge_requirements (challenge_id, required_challenge_id)
SELECT sqlc.arg(challenge_id)::uuid, unnest(sqlc.arg(required_challenge_ids)::uuid[])
ON CONFLICT (challenge_id, required_challenge_id) DO NOTHING;

-- name: GetAllChallengeRequirements :many
SELECT challenge_id, required_challenge_id FROM challenge_requirements ORDER BY challenge_id, required_challenge_id;
