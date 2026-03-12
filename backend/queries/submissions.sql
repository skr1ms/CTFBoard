-- name: CreateSubmission :exec
INSERT INTO submissions (id, user_id, team_id, challenge_id, submitted_flag, is_correct, ip, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetSubmissionsByChallenge :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
       u.username, COALESCE(t.name, '') AS team_name
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubmissionsByChallengeFrozen :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
       u.username, COALESCE(t.name, '') AS team_name
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.created_at <= $2 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetSubmissionsByUser :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
       c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN challenges c ON c.id = s.challenge_id
WHERE s.user_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubmissionsByUserFrozen :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
       c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN challenges c ON c.id = s.challenge_id
WHERE s.user_id = $1 AND s.created_at <= $2 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetSubmissionsByTeam :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
       u.username, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.team_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubmissionsByTeamFrozen :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
       u.username, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.team_id = $1 AND s.created_at <= $2 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetAllSubmissions :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
       u.username, COALESCE(t.name, '') AS team_name, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAllSubmissionsFrozen :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
       u.username, COALESCE(t.name, '') AS team_name, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.created_at <= $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSubmissionsByChallenge :one
SELECT COUNT(*) FROM submissions WHERE challenge_id = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: CountSubmissionsByChallengeFrozen :one
SELECT COUNT(*) FROM submissions WHERE challenge_id = $1 AND created_at <= $2 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: CountSubmissionsByUser :one
SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: CountSubmissionsByUserFrozen :one
SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND created_at <= $2 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: CountSubmissionsByTeam :one
SELECT COUNT(*) FROM submissions WHERE team_id = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: CountSubmissionsByTeamFrozen :one
SELECT COUNT(*) FROM submissions WHERE team_id = $1 AND created_at <= $2 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: CountAllSubmissions :one
SELECT COUNT(*) FROM submissions WHERE banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: CountAllSubmissionsFrozen :one
SELECT COUNT(*) FROM submissions WHERE created_at <= $1 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: CountFailedSubmissionsByIP :one
SELECT COUNT(*) FROM submissions WHERE ip = $1 AND is_correct = FALSE AND created_at > $2 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: GetSubmissionStats :one
SELECT 
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE is_correct = TRUE) AS correct,
    COUNT(*) FILTER (WHERE is_correct = FALSE) AS incorrect
FROM submissions
WHERE challenge_id = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: GetSubmissionStatsFrozen :one
SELECT 
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE is_correct = TRUE) AS correct,
    COUNT(*) FILTER (WHERE is_correct = FALSE) AS incorrect
FROM submissions
WHERE challenge_id = $1 AND created_at <= $2 AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: GetFailsByUserID :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
       c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN challenges c ON c.id = s.challenge_id
WHERE s.user_id = $1 AND s.is_correct = FALSE AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountFailsByUserID :one
SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND is_correct = FALSE AND banned_team_id IS NULL AND banned_user_id IS NULL;

-- name: GetFailsByTeamID :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
       u.username, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.team_id = $1 AND s.is_correct = FALSE AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubmissionByID :one
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at, s.banned_user_id,
    u.username, COALESCE(t.name, '') AS team_name, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL;

-- name: GetSubmissionByIDForUpdate :one
SELECT id, user_id, team_id, challenge_id, submitted_flag, is_correct, ip, created_at, banned_team_id, banned_user_id
FROM submissions
WHERE id = $1
FOR UPDATE;

-- name: SoftBanSubmissionsByTeamID :exec
UPDATE submissions SET banned_team_id = team_id WHERE team_id = $1 AND banned_team_id IS NULL;

-- name: RestoreSubmissionsByBannedTeamID :exec
UPDATE submissions SET banned_team_id = NULL WHERE banned_team_id = $1;

-- name: SoftBanSubmissionsByUserID :exec
UPDATE submissions SET banned_user_id = user_id WHERE user_id = $1 AND banned_user_id IS NULL;

-- name: RestoreSubmissionsByBannedUserID :exec
UPDATE submissions SET banned_user_id = NULL WHERE banned_user_id = $1;

-- name: UpdateSubmission :exec
UPDATE submissions SET is_correct = $2 WHERE id = $1;

-- name: DeleteSubmission :exec
DELETE FROM submissions WHERE id = $1;

-- name: DeleteSubmissionsByTeamID :exec
DELETE FROM submissions WHERE team_id = $1;

-- name: CountFailsByTeamID :one
SELECT COUNT(*) FROM submissions WHERE team_id = $1 AND is_correct = FALSE AND banned_team_id IS NULL AND banned_user_id IS NULL;
