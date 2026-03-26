-- name: CreateSolve :exec
INSERT INTO solves (id, user_id, team_id, challenge_id, solved_at, points_at_solve)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetSolveByID :one
SELECT id, user_id, team_id, challenge_id, solved_at, points_at_solve, banned_team_id, banned_user_id
FROM solves
WHERE id = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: GetSolveByTeamAndChallenge :one
SELECT id, user_id, team_id, challenge_id, solved_at, points_at_solve, banned_team_id, banned_user_id
FROM solves
WHERE team_id = $1 AND challenge_id = $2 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: GetSolveByTeamAndChallengeForUpdate :one
SELECT id, user_id, team_id, challenge_id, solved_at, points_at_solve, banned_team_id, banned_user_id
FROM solves
WHERE team_id = $1 AND challenge_id = $2 AND banned_team_id IS NULL AND banned_user_id IS NULL
FOR UPDATE;

-- name: GetSolvedChallengeIDsByTeam :many
SELECT challenge_id
FROM solves
WHERE team_id = $1 AND challenge_id = ANY($2::uuid[]) AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: DeleteSolvesByTeamID :exec
DELETE FROM solves WHERE team_id = $1;

-- name: SoftBanSolvesByTeamID :exec
UPDATE solves SET banned_team_id = team_id WHERE team_id = $1 AND banned_team_id IS NULL;

-- name: RestoreSolvesByBannedTeamID :exec
UPDATE solves SET banned_team_id = NULL WHERE banned_team_id = $1;

-- name: SoftBanSolvesByTeamIDAndUserID :exec
UPDATE solves SET banned_user_id = user_id WHERE team_id = $1 AND user_id = $2 AND banned_user_id IS NULL;

-- name: RestoreSolvesByBannedUserID :exec
UPDATE solves SET banned_user_id = NULL WHERE banned_user_id = $1;

-- name: DeleteSolveByTeamAndChallenge :exec
DELETE FROM solves WHERE team_id = $1 AND challenge_id = $2;

-- name: GetSolvesByUserID :many
SELECT id, user_id, team_id, challenge_id, solved_at, points_at_solve, banned_team_id, banned_user_id
FROM solves
WHERE user_id = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL
ORDER BY solved_at DESC;

-- name: GetAllSolves :many
SELECT id, user_id, team_id, challenge_id, solved_at, points_at_solve, banned_team_id, banned_user_id
FROM solves
WHERE banned_team_id IS NULL AND banned_user_id IS NULL
ORDER BY solved_at ASC;

-- name: GetSolvesForBackup :many
SELECT id, user_id, team_id, challenge_id, solved_at, points_at_solve, banned_team_id, banned_user_id
FROM solves
ORDER BY solved_at ASC;

-- name: GetScoreboard :many
SELECT
    t.id AS team_id,
    t.name AS team_name,
    COALESCE(solve_points.points, 0) + COALESCE(award_points.total, 0) AS points,
    solve_points.last_solved AS solved_at
FROM teams t
LEFT JOIN (
    SELECT s.team_id, SUM(s.points_at_solve)::int AS points, MAX(s.solved_at) AS last_solved
    FROM solves s
    JOIN challenges c ON c.id = s.challenge_id AND c.state IN ('visible', 'locked')
    WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
    GROUP BY s.team_id
) solve_points ON solve_points.team_id = t.id
LEFT JOIN (
    SELECT team_id, SUM(value)::int AS total
    FROM awards
    WHERE banned_team_id IS NULL
    GROUP BY team_id
) award_points ON award_points.team_id = t.id
WHERE t.is_banned = false AND t.is_hidden = false AND t.deleted_at IS NULL
ORDER BY points DESC, COALESCE(solve_points.last_solved, '9999-12-31'::timestamp) ASC
LIMIT 10000;

-- name: GetScoreboardByBracket :many
SELECT
    t.id AS team_id,
    t.name AS team_name,
    COALESCE(solve_points.points, 0) + COALESCE(award_points.total, 0) AS points,
    solve_points.last_solved AS solved_at
FROM teams t
LEFT JOIN (
    SELECT s.team_id, SUM(s.points_at_solve)::int AS points, MAX(s.solved_at) AS last_solved
    FROM solves s
    JOIN challenges c ON c.id = s.challenge_id AND c.state IN ('visible', 'locked')
    WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
    GROUP BY s.team_id
) solve_points ON solve_points.team_id = t.id
LEFT JOIN (
    SELECT team_id, SUM(value)::int AS total
    FROM awards
    WHERE banned_team_id IS NULL
    GROUP BY team_id
) award_points ON award_points.team_id = t.id
WHERE t.is_banned = false AND t.is_hidden = false AND t.deleted_at IS NULL
  AND (sqlc.narg('bracket_id')::uuid IS NULL OR t.bracket_id = sqlc.narg('bracket_id'))
ORDER BY points DESC, COALESCE(solve_points.last_solved, '9999-12-31'::timestamp) ASC
LIMIT 10000;

-- name: GetScoreboardByBracketFrozen :many
SELECT
    t.id AS team_id,
    t.name AS team_name,
    COALESCE(solve_points.points, 0) + COALESCE(award_points.total, 0) AS points,
    solve_points.last_solved AS solved_at
FROM teams t
LEFT JOIN (
    SELECT s.team_id, SUM(s.points_at_solve)::int AS points, MAX(s.solved_at) AS last_solved
    FROM solves s
    JOIN challenges c ON c.id = s.challenge_id AND c.state IN ('visible', 'locked')
    WHERE s.solved_at <= $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
    GROUP BY s.team_id
) solve_points ON solve_points.team_id = t.id
LEFT JOIN (
    SELECT team_id, SUM(value)::int AS total
    FROM awards
    WHERE awards.created_at <= $1 AND awards.banned_team_id IS NULL
    GROUP BY team_id
) award_points ON award_points.team_id = t.id
WHERE t.is_banned = false AND t.is_hidden = false AND t.deleted_at IS NULL
  AND (sqlc.narg('bracket_id')::uuid IS NULL OR t.bracket_id = sqlc.narg('bracket_id'))
ORDER BY points DESC, COALESCE(solve_points.last_solved, '9999-12-31'::timestamp) ASC
LIMIT 10000;

-- name: GetTeamScore :one
SELECT
    COALESCE((
        SELECT SUM(s.points_at_solve) FROM solves s
        JOIN challenges c ON c.id = s.challenge_id AND c.state IN ('visible', 'locked')
        WHERE s.team_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
    ), 0)::int +
    COALESCE((
        SELECT SUM(value) FROM awards WHERE team_id = $1 AND banned_team_id IS NULL
    ), 0)::int AS total;

-- name: GetFirstBlood :one
SELECT s.user_id, u.username, s.team_id, t.name AS team_name, s.solved_at
FROM solves s
JOIN users u ON u.id = s.user_id
JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
  AND t.is_banned = false AND t.is_hidden = false AND t.deleted_at IS NULL
ORDER BY s.solved_at ASC
LIMIT 1;

-- name: GetFirstBloodFrozen :one
SELECT s.user_id, u.username, s.team_id, t.name AS team_name, s.solved_at
FROM solves s
JOIN users u ON u.id = s.user_id
JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.solved_at <= $2
  AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
  AND t.is_banned = false AND t.is_hidden = false AND t.deleted_at IS NULL
ORDER BY s.solved_at ASC
LIMIT 1;

-- name: GetSolvesByChallengeID :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.solved_at,
       u.username, t.name AS team_name
FROM solves s
JOIN users u ON u.id = s.user_id
JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
  AND t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
ORDER BY s.solved_at ASC;

-- name: GetSolvesByChallengeIDFrozen :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.solved_at,
       u.username, t.name AS team_name
FROM solves s
JOIN users u ON u.id = s.user_id
JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.solved_at <= $2
  AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
  AND t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
ORDER BY s.solved_at ASC;

-- name: GetSolveCountsFrozen :many
SELECT s.challenge_id, COUNT(*)::int AS solve_count
FROM solves s
JOIN teams t ON t.id = s.team_id
JOIN challenges c ON c.id = s.challenge_id AND c.state IN ('visible', 'locked')
WHERE s.solved_at <= $1
  AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
  AND t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
GROUP BY s.challenge_id;

-- name: GetSolvesByUserIDWithDetails :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.solved_at,
       c.title AS challenge_title, c.category AS challenge_category, c.points AS challenge_points
FROM solves s
JOIN challenges c ON c.id = s.challenge_id
WHERE s.user_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL AND c.state IN ('visible', 'locked')
ORDER BY s.solved_at DESC;

-- name: GetSolvesByTeamIDWithDetails :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.solved_at,
       u.username, c.title AS challenge_title, c.category AS challenge_category, c.points AS challenge_points
FROM solves s
JOIN users u ON u.id = s.user_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.team_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL AND c.state IN ('visible', 'locked')
ORDER BY s.solved_at DESC;

-- name: GetSolvesForPointsRecalc :many
SELECT s.id, s.challenge_id, s.solved_at,
       c.initial_value, c.min_value, c.decay
FROM solves s
JOIN challenges c ON c.id = s.challenge_id
JOIN teams t ON t.id = s.team_id AND t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
WHERE s.challenge_id = ANY($1::uuid[]) AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.challenge_id, s.solved_at ASC;

-- name: BatchUpdateSolvePoints :exec
UPDATE solves AS s SET points_at_solve = v.points
FROM (SELECT unnest($1::uuid[]) AS id, unnest($2::int[]) AS points) AS v
WHERE s.id = v.id;
