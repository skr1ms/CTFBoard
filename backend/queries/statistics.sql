-- name: CountUsers :one
SELECT COUNT(*)::int FROM users;

-- name: CountUsersFrozen :one
SELECT COUNT(*)::int FROM users WHERE created_at <= $1;

-- name: CountTeams :one
SELECT COUNT(*)::int FROM teams WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false;

-- name: CountTeamsFrozen :one
SELECT COUNT(*)::int FROM teams WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false AND created_at <= $1;

-- name: CountChallenges :one
SELECT COUNT(*)::int FROM challenges;

-- name: CountSolves :one
SELECT COUNT(*)::int FROM solves WHERE banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: CountSolvesFrozen :one
SELECT COUNT(*)::int FROM solves WHERE solved_at <= $1 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: GetChallengeStats :many
SELECT id, title, category, points, solve_count
FROM challenges
ORDER BY solve_count DESC;

-- name: GetChallengeStatsFrozen :many
SELECT c.id, c.title, c.category, c.points, COUNT(s.id)::int AS solve_count
FROM challenges c
LEFT JOIN solves s ON s.challenge_id = c.id AND s.solved_at <= $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
GROUP BY c.id, c.title, c.category, c.points
ORDER BY solve_count DESC;

-- name: GetChallengeDetailChallenge :one
SELECT c.id, c.title, c.category, c.points, c.solve_count,
    (SELECT COUNT(*)::int FROM teams WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false) AS total_teams
FROM challenges c
WHERE c.id = $1;

-- name: GetChallengeDetailChallengeFrozen :one
SELECT c.id, c.title, c.category, c.points,
    (SELECT COUNT(*)::int FROM solves WHERE challenge_id = c.id AND solved_at <= $2 AND banned_team_id IS NULL AND banned_user_id IS NULL) AS solve_count,
    (SELECT COUNT(*)::int FROM teams WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false) AS total_teams
FROM challenges c
WHERE c.id = $1;

-- name: GetChallengeDetailSolves :many
SELECT s.team_id, t.name AS team_name, s.solved_at
FROM solves s
JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
  AND t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
ORDER BY s.solved_at ASC;

-- name: GetChallengeDetailSolvesFrozen :many
SELECT s.team_id, t.name AS team_name, s.solved_at
FROM solves s
JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.solved_at <= $2
  AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
  AND t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
ORDER BY s.solved_at ASC;

-- name: GetChallengeSolvePercentages :many
WITH total AS (
    SELECT COUNT(*)::int AS n
    FROM teams
    WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false
)
SELECT c.id, c.title, c.category, c.solve_count,
    total.n AS total_teams,
    CASE WHEN total.n = 0 THEN 0
        ELSE ROUND((c.solve_count::numeric / total.n::numeric) * 100, 2)
    END AS percentage
FROM challenges c
CROSS JOIN total
WHERE c.is_hidden = false
ORDER BY percentage DESC;

-- name: GetChallengeSolvePercentagesFrozen :many
WITH total AS (
    SELECT COUNT(*)::int AS n
    FROM teams
    WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false
)
SELECT c.id, c.title, c.category,
    COUNT(s.id)::int AS solve_count,
    total.n AS total_teams,
    CASE WHEN total.n = 0 THEN 0
        ELSE ROUND((COUNT(s.id)::numeric / total.n::numeric) * 100, 2)
    END AS percentage
FROM challenges c
CROSS JOIN total
LEFT JOIN solves s ON s.challenge_id = c.id AND s.solved_at <= $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
WHERE c.is_hidden = false
GROUP BY c.id, c.title, c.category, total.n
ORDER BY percentage DESC;

-- name: GetScoreDistribution :many
WITH buckets AS (
    SELECT CASE
        WHEN score = 0 THEN '0'
        WHEN score <= 100 THEN '1-100'
        WHEN score <= 250 THEN '101-250'
        WHEN score <= 500 THEN '251-500'
        WHEN score <= 1000 THEN '501-1000'
        WHEN score <= 2500 THEN '1001-2500'
        ELSE '2500+'
    END AS bucket,
    CASE
        WHEN score = 0 THEN 0
        WHEN score <= 100 THEN 1
        WHEN score <= 250 THEN 2
        WHEN score <= 500 THEN 3
        WHEN score <= 1000 THEN 4
        WHEN score <= 2500 THEN 5
        ELSE 6
    END AS bucket_order
    FROM (
        SELECT COALESCE(SUM(s.points_at_solve), 0)::int AS score
        FROM teams t
        LEFT JOIN (solves s JOIN challenges c ON c.id = s.challenge_id AND c.is_hidden = false AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL) ON s.team_id = t.id
        WHERE t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
        GROUP BY t.id
    ) scores
)
SELECT bucket, COUNT(*)::int AS count
FROM buckets
GROUP BY bucket, bucket_order
ORDER BY bucket_order;

-- name: GetScoreDistributionFrozen :many
WITH buckets AS (
    SELECT CASE
        WHEN score = 0 THEN '0'
        WHEN score <= 100 THEN '1-100'
        WHEN score <= 250 THEN '101-250'
        WHEN score <= 500 THEN '251-500'
        WHEN score <= 1000 THEN '501-1000'
        WHEN score <= 2500 THEN '1001-2500'
        ELSE '2500+'
    END AS bucket,
    CASE
        WHEN score = 0 THEN 0
        WHEN score <= 100 THEN 1
        WHEN score <= 250 THEN 2
        WHEN score <= 500 THEN 3
        WHEN score <= 1000 THEN 4
        WHEN score <= 2500 THEN 5
        ELSE 6
    END AS bucket_order
    FROM (
        SELECT COALESCE(SUM(s.points_at_solve), 0)::int AS score
        FROM teams t
        LEFT JOIN (solves s JOIN challenges c ON c.id = s.challenge_id AND c.is_hidden = false AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL) ON s.team_id = t.id AND s.solved_at <= $1
        WHERE t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
        GROUP BY t.id
    ) scores
)
SELECT bucket, COUNT(*)::int AS count
FROM buckets
GROUP BY bucket, bucket_order
ORDER BY bucket_order;

-- name: GetSubmissionTimeSeries :many
SELECT DATE(created_at) AS date,
    COUNT(*) FILTER (WHERE is_correct = true)::int AS correct,
    COUNT(*) FILTER (WHERE is_correct = false)::int AS incorrect
FROM submissions
WHERE banned_team_id IS NULL AND banned_user_id IS NULL
GROUP BY DATE(created_at)
ORDER BY date;

-- name: GetSubmissionTimeSeriesFrozen :many
SELECT DATE(created_at) AS date,
    COUNT(*) FILTER (WHERE is_correct = true)::int AS correct,
    COUNT(*) FILTER (WHERE is_correct = false)::int AS incorrect
FROM submissions
WHERE created_at <= $1 AND banned_team_id IS NULL AND banned_user_id IS NULL
GROUP BY DATE(created_at)
ORDER BY date;

-- name: GetSubmissionTimeSeriesByType :many
SELECT DATE(created_at) AS date, COUNT(*)::int AS count
FROM submissions
WHERE is_correct = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL
GROUP BY DATE(created_at)
ORDER BY date;

-- name: GetSubmissionTimeSeriesByTypeFrozen :many
SELECT DATE(created_at) AS date, COUNT(*)::int AS count
FROM submissions
WHERE is_correct = $1 AND created_at <= $2 AND banned_team_id IS NULL AND banned_user_id IS NULL
GROUP BY DATE(created_at)
ORDER BY date;

-- name: GetTeamRegistrationTimeSeries :many
SELECT DATE(created_at) AS date, COUNT(*)::int AS count
FROM teams
WHERE deleted_at IS NULL
GROUP BY DATE(created_at)
ORDER BY date;

-- name: GetUserRegistrationTimeSeries :many
SELECT DATE(created_at) AS date, COUNT(*)::int AS count
FROM users
GROUP BY DATE(created_at)
ORDER BY date;

-- name: GetScoreboardHistory :many
WITH top_teams AS (
    SELECT t.id, t.name
    FROM teams t
    LEFT JOIN (
        SELECT s.team_id, SUM(s.points_at_solve)::int AS total
        FROM solves s
        JOIN challenges c ON c.id = s.challenge_id AND c.is_hidden = false
        WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
        GROUP BY s.team_id
    ) sp ON sp.team_id = t.id
    LEFT JOIN (
        SELECT team_id, SUM(value)::int AS total
        FROM awards
        WHERE banned_team_id IS NULL
        GROUP BY team_id
    ) ap ON ap.team_id = t.id
    WHERE t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
    ORDER BY COALESCE(sp.total, 0) + COALESCE(ap.total, 0) DESC, t.id
    LIMIT $1
),
events AS (
    SELECT s.team_id, s.solved_at AS event_time, s.points_at_solve AS delta
    FROM solves s
    JOIN challenges c ON c.id = s.challenge_id AND c.is_hidden = false
    WHERE s.team_id IN (SELECT id FROM top_teams) AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
    UNION ALL
    SELECT a.team_id, a.created_at AS event_time, a.value AS delta
    FROM awards a
    WHERE a.team_id IN (SELECT id FROM top_teams) AND a.banned_team_id IS NULL
)
SELECT e.team_id, tt.name AS team_name, SUM(e.delta) OVER (PARTITION BY e.team_id ORDER BY e.event_time)::int AS points, e.event_time AS timestamp
FROM events e
JOIN top_teams tt ON e.team_id = tt.id
ORDER BY e.team_id, e.event_time;

-- name: GetScoreboardHistoryFrozen :many
WITH top_teams AS (
    SELECT t.id, t.name
    FROM teams t
    LEFT JOIN (
        SELECT sv.team_id, SUM(sv.points_at_solve)::int AS total
        FROM solves sv
        JOIN challenges c ON c.id = sv.challenge_id AND c.is_hidden = false
        WHERE sv.solved_at <= $2 AND sv.banned_team_id IS NULL AND sv.banned_user_id IS NULL
        GROUP BY sv.team_id
    ) sp ON sp.team_id = t.id
    LEFT JOIN (
        SELECT aw.team_id, SUM(aw.value)::int AS total
        FROM awards aw
        WHERE aw.created_at <= $2 AND aw.banned_team_id IS NULL
        GROUP BY aw.team_id
    ) ap ON ap.team_id = t.id
    WHERE t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
    ORDER BY COALESCE(sp.total, 0) + COALESCE(ap.total, 0) DESC, t.id
    LIMIT $1
),
events AS (
    SELECT s.team_id, s.solved_at AS event_time, s.points_at_solve AS delta
    FROM solves s
    JOIN challenges c ON c.id = s.challenge_id AND c.is_hidden = false
    WHERE s.team_id IN (SELECT id FROM top_teams)
      AND s.solved_at <= $2 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
    UNION ALL
    SELECT a.team_id, a.created_at AS event_time, a.value AS delta
    FROM awards a
    WHERE a.team_id IN (SELECT id FROM top_teams)
      AND a.created_at <= $2 AND a.banned_team_id IS NULL
)
SELECT e.team_id, tt.name AS team_name, SUM(e.delta) OVER (PARTITION BY e.team_id ORDER BY e.event_time)::int AS points, e.event_time AS timestamp
FROM events e
JOIN top_teams tt ON e.team_id = tt.id
ORDER BY e.team_id, e.event_time;

-- name: GetSolveMatrix :many
-- CROSS JOIN teams x challenges can be heavy for large datasets; used by admin statistics only and cached.
SELECT 
    t.id AS team_id,
    t.name AS team_name,
    c.id AS challenge_id,
    c.title AS challenge_title,
    c.category AS challenge_category,
    CASE WHEN s.id IS NOT NULL THEN true ELSE false END AS solved,
    s.solved_at
FROM teams t
CROSS JOIN challenges c
LEFT JOIN solves s ON s.team_id = t.id AND s.challenge_id = c.id AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
WHERE t.deleted_at IS NULL 
    AND t.is_banned = false 
    AND t.is_hidden = false
    AND c.is_hidden = false
ORDER BY t.name, c.category, c.title;

-- name: GetSolveMatrixFrozen :many
SELECT 
    t.id AS team_id,
    t.name AS team_name,
    c.id AS challenge_id,
    c.title AS challenge_title,
    c.category AS challenge_category,
    CASE WHEN s.id IS NOT NULL THEN true ELSE false END AS solved,
    s.solved_at
FROM teams t
CROSS JOIN challenges c
LEFT JOIN solves s ON s.team_id = t.id AND s.challenge_id = c.id AND s.solved_at <= $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
WHERE t.deleted_at IS NULL 
    AND t.is_banned = false 
    AND t.is_hidden = false
    AND c.is_hidden = false
ORDER BY t.name, c.category, c.title;
