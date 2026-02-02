-- name: CountUsers :one
SELECT COUNT(*)::int FROM users;

-- name: CountTeams :one
SELECT COUNT(*)::int FROM teams WHERE deleted_at IS NULL;

-- name: CountChallenges :one
SELECT COUNT(*)::int FROM challenges;

-- name: CountSolves :one
SELECT COUNT(*)::int FROM solves;

-- name: GetChallengeStats :many
SELECT id, title, category, points, solve_count
FROM challenges
ORDER BY solve_count DESC;

-- name: GetChallengeDetailChallenge :one
SELECT c.id, c.title, c.category, c.points, c.solve_count,
    (SELECT COUNT(*)::int FROM teams WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false) AS total_teams
FROM challenges c
WHERE c.id = $1;

-- name: GetChallengeDetailSolves :many
SELECT s.team_id, t.name AS team_name, s.solved_at
FROM solves s
JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
ORDER BY s.solved_at ASC;

-- name: GetScoreboardHistory :many
WITH top_teams AS (
    SELECT t.id, t.name
    FROM teams t
    LEFT JOIN solves s ON s.team_id = t.id
    LEFT JOIN challenges c ON s.challenge_id = c.id
    LEFT JOIN awards a ON a.team_id = t.id
    WHERE t.deleted_at IS NULL
    GROUP BY t.id
    ORDER BY COALESCE(SUM(c.points), 0) + COALESCE(SUM(a.value), 0) DESC
    LIMIT $1
),
events AS (
    SELECT s.team_id, s.solved_at AS event_time, c.points AS delta
    FROM solves s
    JOIN challenges c ON s.challenge_id = c.id
    WHERE s.team_id IN (SELECT id FROM top_teams)
    UNION ALL
    SELECT a.team_id, a.created_at AS event_time, a.value AS delta
    FROM awards a
    WHERE a.team_id IN (SELECT id FROM top_teams)
)
SELECT e.team_id, tt.name AS team_name, SUM(e.delta) OVER (PARTITION BY e.team_id ORDER BY e.event_time)::int AS points, e.event_time AS timestamp
FROM events e
JOIN top_teams tt ON e.team_id = tt.id
ORDER BY e.team_id, e.event_time;
