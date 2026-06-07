-- name: CountUsers :one
SELECT COUNT(*)::int FROM users
WHERE is_banned = false
  AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR created_at <= sqlc.narg('freeze_time'));

-- name: CountTeams :one
SELECT COUNT(*)::int FROM teams WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false
  AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR created_at <= sqlc.narg('freeze_time'));

-- name: CountChallenges :one
SELECT COUNT(*)::int FROM challenges WHERE state IN ('visible', 'locked');

-- name: CountSolves :one
SELECT COUNT(*)::int FROM solves WHERE banned_team_id IS NULL AND banned_user_id IS NULL
  AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR solved_at <= sqlc.narg('freeze_time'));

-- name: GetGeneralStats :one
SELECT
    (
        SELECT COUNT(*)::int
        FROM users
        WHERE is_banned = false
          AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR created_at <= sqlc.narg('freeze_time'))
    ) AS user_count,
    (
        SELECT COUNT(*)::int
        FROM teams
        WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false
          AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR created_at <= sqlc.narg('freeze_time'))
    ) AS team_count,
    (
        SELECT COUNT(*)::int
        FROM challenges
        WHERE state IN ('visible', 'locked')
    ) AS challenge_count,
    (
        SELECT COUNT(*)::int
        FROM solves
        WHERE banned_team_id IS NULL AND banned_user_id IS NULL
          AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR solved_at <= sqlc.narg('freeze_time'))
    ) AS solve_count;

-- name: GetChallengeStats :many
SELECT c.id, c.title, c.category, c.points,
    COUNT(s.id)::int AS solve_count
FROM challenges c
LEFT JOIN solves s ON s.challenge_id = c.id AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
    AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
WHERE c.state IN ('visible', 'locked')
GROUP BY c.id, c.title, c.category, c.points
ORDER BY solve_count DESC;

-- name: GetChallengeDetailChallenge :one
SELECT c.id, c.title, c.category, c.points,
    (SELECT COUNT(*)::int FROM solves
     WHERE challenge_id = c.id AND banned_team_id IS NULL AND banned_user_id IS NULL
       AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR solved_at <= sqlc.narg('freeze_time'))
    ) AS solve_count,
    (SELECT COUNT(*)::int FROM teams WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false) AS total_teams
FROM challenges c
WHERE c.id = $1;

-- name: GetChallengeDetailSolves :many
SELECT s.team_id, t.name AS team_name, s.solved_at
FROM solves s
JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
  AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
  AND t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
ORDER BY s.solved_at ASC;

-- name: GetChallengeSolvePercentages :many
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
LEFT JOIN solves s ON s.challenge_id = c.id AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
    AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
WHERE c.state IN ('visible', 'locked')
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
        SELECT COALESCE(sp.points, 0) + COALESCE(ap.total, 0) AS score
        FROM teams t
        LEFT JOIN (
            SELECT s.team_id, SUM(s.points_at_solve)::int AS points
            FROM solves s
            JOIN challenges c ON c.id = s.challenge_id AND c.state IN ('visible', 'locked')
            WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
              AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
            GROUP BY s.team_id
        ) sp ON sp.team_id = t.id
        LEFT JOIN (
            SELECT team_id, SUM(value)::int AS total
            FROM awards
            WHERE banned_team_id IS NULL
              AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR created_at <= sqlc.narg('freeze_time'))
            GROUP BY team_id
        ) ap ON ap.team_id = t.id
        WHERE t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
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
WHERE banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect')
  AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR created_at <= sqlc.narg('freeze_time'))
GROUP BY DATE(created_at)
ORDER BY date;

-- name: GetSubmissionTimeSeriesByType :many
SELECT DATE(created_at) AS date, COUNT(*)::int AS count
FROM submissions
WHERE is_correct = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect')
  AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR created_at <= sqlc.narg('freeze_time'))
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
        JOIN challenges c ON c.id = s.challenge_id AND c.state IN ('visible', 'locked')
        WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
          AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
        GROUP BY s.team_id
    ) sp ON sp.team_id = t.id
    LEFT JOIN (
        SELECT team_id, SUM(value)::int AS total
        FROM awards
        WHERE banned_team_id IS NULL
          AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR created_at <= sqlc.narg('freeze_time'))
        GROUP BY team_id
    ) ap ON ap.team_id = t.id
    WHERE t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false
    ORDER BY COALESCE(sp.total, 0) + COALESCE(ap.total, 0) DESC, t.id
    LIMIT $1
),
events AS (
    SELECT s.team_id, s.solved_at AS event_time, s.points_at_solve AS delta
    FROM solves s
    JOIN challenges c ON c.id = s.challenge_id AND c.state IN ('visible', 'locked')
    WHERE s.team_id IN (SELECT id FROM top_teams)
      AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
    UNION ALL
    SELECT a.team_id, a.created_at AS event_time, a.value AS delta
    FROM awards a
    WHERE a.team_id IN (SELECT id FROM top_teams) AND a.banned_team_id IS NULL
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR a.created_at <= sqlc.narg('freeze_time'))
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
LEFT JOIN solves s ON s.team_id = t.id AND s.challenge_id = c.id
    AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
    AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
WHERE t.deleted_at IS NULL 
    AND t.is_banned = false 
    AND t.is_hidden = false
    AND c.state IN ('visible', 'locked')
ORDER BY t.name, c.category, c.title;

-- name: GetFunnelChallengeRows :many
WITH active_challenges AS (
    SELECT id, title, category
    FROM challenges
    WHERE state IN ('visible', 'locked')
),
active_teams AS (
    SELECT id
    FROM teams
    WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false
),
open_counts AS (
    SELECT co.challenge_id, COUNT(DISTINCT COALESCE(co.team_id, u.team_id))::int AS opened_count
    FROM challenge_opens co
    JOIN users u ON u.id = co.user_id AND u.is_banned = false
    JOIN active_teams t ON t.id = COALESCE(co.team_id, u.team_id)
    WHERE (sqlc.narg('freeze_time')::timestamptz IS NULL OR co.opened_at <= sqlc.narg('freeze_time'))
    GROUP BY co.challenge_id
),
attempt_counts AS (
    SELECT s.challenge_id, COUNT(DISTINCT s.team_id)::int AS attempted_count
    FROM submissions s
    JOIN active_teams t ON t.id = s.team_id
    WHERE s.team_id IS NOT NULL
      AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND s.submission_type IN ('correct', 'incorrect')
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.created_at <= sqlc.narg('freeze_time'))
    GROUP BY s.challenge_id
),
solve_counts AS (
    SELECT s.challenge_id, COUNT(DISTINCT s.team_id)::int AS solved_count
    FROM solves s
    JOIN active_teams t ON t.id = s.team_id
    WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
    GROUP BY s.challenge_id
)
SELECT c.id AS challenge_id,
       c.title AS challenge_title,
       c.category AS challenge_category,
       COALESCE(oc.opened_count, 0)::int AS opened_count,
       COALESCE(ac.attempted_count, 0)::int AS attempted_count,
       COALESCE(sc.solved_count, 0)::int AS solved_count
FROM active_challenges c
LEFT JOIN open_counts oc ON oc.challenge_id = c.id
LEFT JOIN attempt_counts ac ON ac.challenge_id = c.id
LEFT JOIN solve_counts sc ON sc.challenge_id = c.id
ORDER BY c.category, c.title, c.id;

-- name: GetFunnelTeamRows :many
WITH active_teams AS (
    SELECT id, name
    FROM teams
    WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false
),
active_challenges AS (
    SELECT id
    FROM challenges
    WHERE state IN ('visible', 'locked')
),
team_opens AS (
    SELECT COALESCE(co.team_id, u.team_id) AS team_id, co.challenge_id, MIN(co.opened_at) AS first_opened_at
    FROM challenge_opens co
    JOIN users u ON u.id = co.user_id AND u.is_banned = false
    JOIN active_teams t ON t.id = COALESCE(co.team_id, u.team_id)
    JOIN active_challenges c ON c.id = co.challenge_id
    WHERE (sqlc.narg('freeze_time')::timestamptz IS NULL OR co.opened_at <= sqlc.narg('freeze_time'))
    GROUP BY COALESCE(co.team_id, u.team_id), co.challenge_id
),
team_attempts AS (
    SELECT s.team_id, s.challenge_id, MIN(s.created_at) AS first_attempted_at
    FROM submissions s
    JOIN active_teams t ON t.id = s.team_id
    JOIN active_challenges c ON c.id = s.challenge_id
    WHERE s.team_id IS NOT NULL
      AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND s.submission_type IN ('correct', 'incorrect')
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.created_at <= sqlc.narg('freeze_time'))
    GROUP BY s.team_id, s.challenge_id
),
team_solves AS (
    SELECT s.team_id, s.challenge_id, MIN(s.solved_at) AS solved_at
    FROM solves s
    JOIN active_teams t ON t.id = s.team_id
    JOIN active_challenges c ON c.id = s.challenge_id
    WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
    GROUP BY s.team_id, s.challenge_id
)
SELECT t.id AS team_id,
       t.name AS team_name,
       COUNT(DISTINCT o.challenge_id)::int AS opened_count,
       COUNT(DISTINCT a.challenge_id)::int AS attempted_count,
       COUNT(DISTINCT sol.challenge_id)::int AS solved_count
FROM active_teams t
LEFT JOIN team_opens o ON o.team_id = t.id
LEFT JOIN team_attempts a ON a.team_id = t.id
LEFT JOIN team_solves sol ON sol.team_id = t.id
GROUP BY t.id, t.name
ORDER BY solved_count DESC, attempted_count DESC, opened_count DESC, t.name, t.id
LIMIT $1;

-- name: GetFunnelTeamCells :many
WITH active_teams AS (
    SELECT id, name
    FROM teams
    WHERE deleted_at IS NULL AND is_banned = false AND is_hidden = false
),
active_challenges AS (
    SELECT id, category, title
    FROM challenges
    WHERE state IN ('visible', 'locked')
),
team_opens AS (
    SELECT COALESCE(co.team_id, u.team_id) AS team_id, co.challenge_id, MIN(co.opened_at) AS first_opened_at
    FROM challenge_opens co
    JOIN users u ON u.id = co.user_id AND u.is_banned = false
    JOIN active_teams t ON t.id = COALESCE(co.team_id, u.team_id)
    JOIN active_challenges c ON c.id = co.challenge_id
    WHERE (sqlc.narg('freeze_time')::timestamptz IS NULL OR co.opened_at <= sqlc.narg('freeze_time'))
    GROUP BY COALESCE(co.team_id, u.team_id), co.challenge_id
),
team_attempts AS (
    SELECT s.team_id, s.challenge_id, MIN(s.created_at) AS first_attempted_at
    FROM submissions s
    JOIN active_teams t ON t.id = s.team_id
    JOIN active_challenges c ON c.id = s.challenge_id
    WHERE s.team_id IS NOT NULL
      AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND s.submission_type IN ('correct', 'incorrect')
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.created_at <= sqlc.narg('freeze_time'))
    GROUP BY s.team_id, s.challenge_id
),
team_solves AS (
    SELECT s.team_id, s.challenge_id, MIN(s.solved_at) AS solved_at
    FROM solves s
    JOIN active_teams t ON t.id = s.team_id
    JOIN active_challenges c ON c.id = s.challenge_id
    WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
    GROUP BY s.team_id, s.challenge_id
),
top_teams AS (
    SELECT t.id, t.name,
           COUNT(DISTINCT o.challenge_id)::int AS opened_count,
           COUNT(DISTINCT a.challenge_id)::int AS attempted_count,
           COUNT(DISTINCT sol.challenge_id)::int AS solved_count
    FROM active_teams t
    LEFT JOIN team_opens o ON o.team_id = t.id
    LEFT JOIN team_attempts a ON a.team_id = t.id
    LEFT JOIN team_solves sol ON sol.team_id = t.id
    GROUP BY t.id, t.name
    ORDER BY solved_count DESC, attempted_count DESC, opened_count DESC, t.name, t.id
    LIMIT $1
)
SELECT tt.id AS team_id,
       c.id AS challenge_id,
       (o.first_opened_at IS NOT NULL)::bool AS opened,
       (a.first_attempted_at IS NOT NULL)::bool AS attempted,
       (sol.solved_at IS NOT NULL)::bool AS solved,
       o.first_opened_at::timestamptz AS first_opened_at,
       a.first_attempted_at::timestamptz AS first_attempted_at,
       sol.solved_at::timestamptz AS solved_at
FROM top_teams tt
CROSS JOIN active_challenges c
LEFT JOIN team_opens o ON o.team_id = tt.id AND o.challenge_id = c.id
LEFT JOIN team_attempts a ON a.team_id = tt.id AND a.challenge_id = c.id
LEFT JOIN team_solves sol ON sol.team_id = tt.id AND sol.challenge_id = c.id
ORDER BY tt.name, c.category, c.title, c.id;

-- name: GetFunnelUserRows :many
WITH active_users AS (
    SELECT u.id, u.username
    FROM users u
    LEFT JOIN teams t ON t.id = u.team_id
    WHERE u.is_banned = false
      AND (u.team_id IS NULL OR (t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false))
),
active_challenges AS (
    SELECT id
    FROM challenges
    WHERE state IN ('visible', 'locked')
),
user_opens AS (
    SELECT co.user_id, co.challenge_id, MIN(co.opened_at) AS first_opened_at
    FROM challenge_opens co
    JOIN active_users u ON u.id = co.user_id
    JOIN active_challenges c ON c.id = co.challenge_id
    WHERE (sqlc.narg('freeze_time')::timestamptz IS NULL OR co.opened_at <= sqlc.narg('freeze_time'))
    GROUP BY co.user_id, co.challenge_id
),
user_attempts AS (
    SELECT s.user_id, s.challenge_id, MIN(s.created_at) AS first_attempted_at
    FROM submissions s
    JOIN active_users u ON u.id = s.user_id
    JOIN active_challenges c ON c.id = s.challenge_id
    WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND s.submission_type IN ('correct', 'incorrect')
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.created_at <= sqlc.narg('freeze_time'))
    GROUP BY s.user_id, s.challenge_id
),
user_solves AS (
    SELECT s.user_id, s.challenge_id, MIN(s.solved_at) AS solved_at
    FROM solves s
    JOIN active_users u ON u.id = s.user_id
    JOIN active_challenges c ON c.id = s.challenge_id
    WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
    GROUP BY s.user_id, s.challenge_id
)
SELECT u.id AS user_id,
       u.username,
       COUNT(DISTINCT o.challenge_id)::int AS opened_count,
       COUNT(DISTINCT a.challenge_id)::int AS attempted_count,
       COUNT(DISTINCT sol.challenge_id)::int AS solved_count
FROM active_users u
LEFT JOIN user_opens o ON o.user_id = u.id
LEFT JOIN user_attempts a ON a.user_id = u.id
LEFT JOIN user_solves sol ON sol.user_id = u.id
GROUP BY u.id, u.username
ORDER BY solved_count DESC, attempted_count DESC, opened_count DESC, u.username, u.id
LIMIT $1;

-- name: GetFunnelUserCells :many
WITH active_users AS (
    SELECT u.id, u.username
    FROM users u
    LEFT JOIN teams t ON t.id = u.team_id
    WHERE u.is_banned = false
      AND (u.team_id IS NULL OR (t.deleted_at IS NULL AND t.is_banned = false AND t.is_hidden = false))
),
active_challenges AS (
    SELECT id, category, title
    FROM challenges
    WHERE state IN ('visible', 'locked')
),
user_opens AS (
    SELECT co.user_id, co.challenge_id, MIN(co.opened_at) AS first_opened_at
    FROM challenge_opens co
    JOIN active_users u ON u.id = co.user_id
    JOIN active_challenges c ON c.id = co.challenge_id
    WHERE (sqlc.narg('freeze_time')::timestamptz IS NULL OR co.opened_at <= sqlc.narg('freeze_time'))
    GROUP BY co.user_id, co.challenge_id
),
user_attempts AS (
    SELECT s.user_id, s.challenge_id, MIN(s.created_at) AS first_attempted_at
    FROM submissions s
    JOIN active_users u ON u.id = s.user_id
    JOIN active_challenges c ON c.id = s.challenge_id
    WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND s.submission_type IN ('correct', 'incorrect')
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.created_at <= sqlc.narg('freeze_time'))
    GROUP BY s.user_id, s.challenge_id
),
user_solves AS (
    SELECT s.user_id, s.challenge_id, MIN(s.solved_at) AS solved_at
    FROM solves s
    JOIN active_users u ON u.id = s.user_id
    JOIN active_challenges c ON c.id = s.challenge_id
    WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
      AND (sqlc.narg('freeze_time')::timestamptz IS NULL OR s.solved_at <= sqlc.narg('freeze_time'))
    GROUP BY s.user_id, s.challenge_id
),
top_users AS (
    SELECT u.id, u.username,
           COUNT(DISTINCT o.challenge_id)::int AS opened_count,
           COUNT(DISTINCT a.challenge_id)::int AS attempted_count,
           COUNT(DISTINCT sol.challenge_id)::int AS solved_count
    FROM active_users u
    LEFT JOIN user_opens o ON o.user_id = u.id
    LEFT JOIN user_attempts a ON a.user_id = u.id
    LEFT JOIN user_solves sol ON sol.user_id = u.id
    GROUP BY u.id, u.username
    ORDER BY solved_count DESC, attempted_count DESC, opened_count DESC, u.username, u.id
    LIMIT $1
)
SELECT tu.id AS user_id,
       c.id AS challenge_id,
       (o.first_opened_at IS NOT NULL)::bool AS opened,
       (a.first_attempted_at IS NOT NULL)::bool AS attempted,
       (sol.solved_at IS NOT NULL)::bool AS solved,
       o.first_opened_at::timestamptz AS first_opened_at,
       a.first_attempted_at::timestamptz AS first_attempted_at,
       sol.solved_at::timestamptz AS solved_at
FROM top_users tu
CROSS JOIN active_challenges c
LEFT JOIN user_opens o ON o.user_id = tu.id AND o.challenge_id = c.id
LEFT JOIN user_attempts a ON a.user_id = tu.id AND a.challenge_id = c.id
LEFT JOIN user_solves sol ON sol.user_id = tu.id AND sol.challenge_id = c.id
ORDER BY tu.username, c.category, c.title, c.id;
