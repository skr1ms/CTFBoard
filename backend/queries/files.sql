-- name: CreateFile :exec
INSERT INTO files (id, type, challenge_id, location, filename, size, sha256, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetFileByID :one
SELECT id, type, challenge_id, location, filename, size, sha256, created_at
FROM files
WHERE id = $1;

-- name: GetFilesByChallengeIDAndType :many
SELECT id, type, challenge_id, location, filename, size, sha256, created_at
FROM files
WHERE challenge_id = $1 AND type = $2
ORDER BY created_at DESC;

-- name: GetAllFiles :many
SELECT id, type, challenge_id, location, filename, size, sha256, created_at
FROM files
ORDER BY created_at DESC;

-- name: DeleteFile :one
DELETE FROM files WHERE id = $1 RETURNING id;
