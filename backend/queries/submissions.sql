-- name: CreateSubmission :exec
INSERT INTO submissions (id, user_id, team_id, challenge_id, submitted_flag, is_correct, ip, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetSubmissionsByChallenge :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at,
       u.username, COALESCE(t.name, '') AS team_name
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
WHERE s.challenge_id = $1
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubmissionsByUser :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at,
       c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN challenges c ON c.id = s.challenge_id
WHERE s.user_id = $1
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetSubmissionsByTeam :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at,
       u.username, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
JOIN challenges c ON c.id = s.challenge_id
WHERE s.team_id = $1
ORDER BY s.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetAllSubmissions :many
SELECT s.id, s.user_id, s.team_id, s.challenge_id, s.submitted_flag, s.is_correct, s.ip, s.created_at,
       u.username, COALESCE(t.name, '') AS team_name, c.title AS challenge_title, c.category AS challenge_category
FROM submissions s
JOIN users u ON u.id = s.user_id
LEFT JOIN teams t ON t.id = s.team_id
JOIN challenges c ON c.id = s.challenge_id
ORDER BY s.created_at DESC
LIMIT $1 OFFSET $2;

-- name: CountSubmissionsByChallenge :one
SELECT COUNT(*) FROM submissions WHERE challenge_id = $1;

-- name: CountSubmissionsByUser :one
SELECT COUNT(*) FROM submissions WHERE user_id = $1;

-- name: CountSubmissionsByTeam :one
SELECT COUNT(*) FROM submissions WHERE team_id = $1;

-- name: CountAllSubmissions :one
SELECT COUNT(*) FROM submissions;

-- name: CountFailedSubmissionsByIP :one
SELECT COUNT(*) FROM submissions WHERE ip = $1 AND is_correct = FALSE AND created_at > $2;

-- name: GetSubmissionStats :one
SELECT 
    COUNT(*) AS total,
    COUNT(*) FILTER (WHERE is_correct = TRUE) AS correct,
    COUNT(*) FILTER (WHERE is_correct = FALSE) AS incorrect
FROM submissions
WHERE challenge_id = $1;
