-- name: CreateSubmission :exec
INSERT INTO submissions (id, user_id, team_id, challenge_id, submitted_flag, is_correct, submission_type, ip, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetSubmissionsByChallenge :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
       u.username, COALESCE(t.name, '') AS team_name
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubmissionsByChallengeFrozen :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
       u.username, COALESCE(t.name, '') AS team_name
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1 AND s.created_at <= $2 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetSubmissionsByUser :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
       c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN challenges c ON c.id = s.challenge_id
WHERE s.user_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubmissionsByUserFrozen :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
       c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN challenges c ON c.id = s.challenge_id
WHERE s.user_id = $1 AND s.created_at <= $2 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetSubmissionsByTeam :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
       u.username, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.team_id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubmissionsByTeamFrozen :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
       u.username, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.team_id = $1 AND s.created_at <= $2 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $3 OFFSET $4;

-- name: GetAllSubmissions :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
       u.username, COALESCE(t.name, '') AS team_name, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAllSubmissionsFrozen :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
       u.username, COALESCE(t.name, '') AS team_name, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.created_at <= $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountSubmissionsByChallenge :one
SELECT COUNT(*) FROM submissions WHERE challenge_id = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: CountSubmissionsByChallengeFrozen :one
SELECT COUNT(*) FROM submissions WHERE challenge_id = $1 AND created_at <= $2 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: CountSubmissionsByUser :one
SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: CountSubmissionsByUserFrozen :one
SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND created_at <= $2 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: CountSubmissionsByTeam :one
SELECT COUNT(*) FROM submissions WHERE team_id = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: CountSubmissionsByTeamAndChallenge :one
-- Counts only correct and incorrect submissions for max_attempts; ratelimited entries are excluded (audit-only, not a real attempt).
SELECT COUNT(*)::bigint FROM submissions WHERE team_id = $1 AND challenge_id = $2 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: CountSubmissionsByTeamFrozen :one
SELECT COUNT(*) FROM submissions WHERE team_id = $1 AND created_at <= $2 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: CountAllSubmissions :one
SELECT COUNT(*) FROM submissions WHERE banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: CountAllSubmissionsFrozen :one
SELECT COUNT(*) FROM submissions WHERE created_at <= $1 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: CountFailedSubmissionsByIP :one
SELECT COUNT(*) FROM submissions WHERE ip = $1 AND is_correct = FALSE AND created_at > $2 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: GetSubmissionStats :one
SELECT 
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE is_correct = TRUE) AS correct,
    COUNT(*) FILTER (WHERE is_correct = FALSE) AS incorrect
FROM submissions
WHERE challenge_id = $1 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: GetSubmissionStatsFrozen :one
SELECT 
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE is_correct = TRUE) AS correct,
    COUNT(*) FILTER (WHERE is_correct = FALSE) AS incorrect
FROM submissions
WHERE challenge_id = $1 AND created_at <= $2 AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: GetFailsByUserID :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
       c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN challenges c ON c.id = s.challenge_id
WHERE s.user_id = $1 AND s.is_correct = FALSE AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL AND s.submission_type IN ('correct', 'incorrect')
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: CountFailsByUserID :one
SELECT COUNT(*) FROM submissions WHERE user_id = $1 AND is_correct = FALSE AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');

-- name: GetFailsByTeamID :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
       u.username, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.team_id = $1 AND s.is_correct = FALSE AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL AND s.submission_type IN ('correct', 'incorrect')
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubmissionByID :one
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.submission_type, s.ip, s.created_at, s.banned_user_id,
    u.username, COALESCE(t.name, '') AS team_name, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.id = $1 AND s.banned_team_id IS NULL AND s.banned_user_id IS NULL;

-- name: GetSubmissionByIDForUpdate :one
SELECT id, user_id, team_id, challenge_id, submitted_flag, is_correct, submission_type, ip, created_at, banned_team_id, banned_user_id
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
UPDATE submissions
SET is_correct = $2,
    submission_type = CASE WHEN $2 THEN 'correct' ELSE 'incorrect' END
WHERE id = $1;

-- name: DeleteSubmission :exec
DELETE FROM submissions WHERE id = $1;

-- name: DeleteSubmissionsByTeamID :exec
DELETE FROM submissions WHERE team_id = $1;

-- name: CountFailsByTeamID :one
SELECT COUNT(*) FROM submissions WHERE team_id = $1 AND is_correct = FALSE AND banned_team_id IS NULL AND banned_user_id IS NULL AND submission_type IN ('correct', 'incorrect');
