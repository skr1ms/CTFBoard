-- name: CreateSolve :exec
INSERT INTO solves (id, user_id, team_id, challenge_id, solved_at)
VALUES ($1, $2, $3, $4, $5);

-- name: GetSolveByID :one
SELECT id, user_id, team_id, challenge_id, solved_at
FROM solves
WHERE id = $1;

-- name: GetSolveByTeamAndChallenge :one
SELECT id, user_id, team_id, challenge_id, solved_at
FROM solves
WHERE team_id = $1 AND challenge_id = $2;

-- name: GetSolveByTeamAndChallengeForUpdate :one
SELECT id, user_id, team_id, challenge_id, solved_at
FROM solves
WHERE team_id = $1 AND challenge_id = $2
FOR UPDATE;

-- name: DeleteSolvesByTeamID :exec
DELETE FROM solves WHERE team_id = $1;

-- name: GetSolvesByUserID :many
SELECT id, user_id, team_id, challenge_id, solved_at
FROM solves
WHERE user_id = $1
ORDER BY solved_at DESC;

-- name: GetAllSolves :many
SELECT id, user_id, team_id, challenge_id, solved_at
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
    SELECT s.team_id, SUM(c.points)::int AS points, MAX(s.solved_at) AS last_solved
    FROM solves s
    JOIN challenges c ON c.id = s.challenge_id
    GROUP BY s.team_id
) solve_points ON solve_points.team_id = t.id
LEFT JOIN (
    SELECT team_id, SUM(value)::int AS total
    FROM awards
    GROUP BY team_id
) award_points ON award_points.team_id = t.id
WHERE t.is_banned = false AND t.is_hidden = false AND t.deleted_at IS NULL
ORDER BY points DESC, COALESCE(solve_points.last_solved, '9999-12-31'::timestamp) ASC;

-- name: GetScoreboardFrozen :many
SELECT
    t.id AS team_id,
    t.name AS team_name,
    COALESCE(solve_points.points, 0) + COALESCE(award_points.total, 0) AS points,
    solve_points.last_solved AS solved_at
FROM teams t
LEFT JOIN (
    SELECT s.team_id, SUM(c.points)::int AS points, MAX(s.solved_at) AS last_solved
    FROM solves s
    JOIN challenges c ON c.id = s.challenge_id
    WHERE s.solved_at <= $1
    GROUP BY s.team_id
) solve_points ON solve_points.team_id = t.id
LEFT JOIN (
    SELECT team_id, SUM(value)::int AS total
    FROM awards
    WHERE awards.created_at <= $2
    GROUP BY team_id
) award_points ON award_points.team_id = t.id
WHERE t.is_banned = false AND t.is_hidden = false AND t.deleted_at IS NULL
ORDER BY points DESC, COALESCE(solve_points.last_solved, '9999-12-31'::timestamp) ASC;

-- name: GetTeamScore :one
SELECT
    COALESCE((
        SELECT SUM(c.points) FROM solves s
        JOIN challenges c ON c.id = s.challenge_id
        WHERE s.team_id = $1
    ), 0)::int +
    COALESCE((
        SELECT SUM(value) FROM awards WHERE team_id = $1
    ), 0)::int AS total;

-- name: GetFirstBlood :one
SELECT s.user_id, u.username, s.team_id, t.name AS team_name, s.solved_at
FROM solves s
JOIN users u ON u.id = s.user_id
JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1
ORDER BY s.solved_at ASC
LIMIT 1;
