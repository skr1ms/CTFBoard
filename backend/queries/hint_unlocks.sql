-- name: CreateHintUnlock :one
INSERT INTO hint_unlocks (id, hint_id, team_id)
VALUES ($1, $2, $3)
ON CONFLICT (team_id, hint_id) DO NOTHING
RETURNING id;

-- name: GetHintUnlockByTeamAndHint :one
SELECT id, hint_id, team_id, unlocked_at
FROM hint_unlocks
WHERE team_id = $1 AND hint_id = $2 AND banned_team_id IS NULL;

-- name: GetHintUnlockByTeamAndHintForUpdate :one
SELECT id, hint_id, team_id, unlocked_at
FROM hint_unlocks
WHERE team_id = $1 AND hint_id = $2 AND banned_team_id IS NULL
FOR UPDATE;

-- name: GetAllHintUnlocksSimple :many
SELECT id, hint_id, team_id, unlocked_at
FROM hint_unlocks
WHERE banned_team_id IS NULL
ORDER BY unlocked_at;

-- name: GetHintUnlocksForBackup :many
SELECT id, hint_id, team_id, unlocked_at, banned_team_id
FROM hint_unlocks
ORDER BY unlocked_at;

-- name: GetAllHintUnlocks :many
SELECT hu.id, hu.hint_id, hu.team_id, hu.unlocked_at,
    h.challenge_id, h.cost AS hint_cost
FROM hint_unlocks hu
JOIN hints h ON h.id = hu.hint_id
WHERE hu.banned_team_id IS NULL
ORDER BY hu.unlocked_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAllHintUnlocks :one
SELECT COUNT(*)::int FROM hint_unlocks WHERE banned_team_id IS NULL;

-- name: CountHintUnlocksByTeamID :one
SELECT COUNT(*)::int FROM hint_unlocks WHERE team_id = $1 AND banned_team_id IS NULL;

-- name: GetUnlockedHintIDs :many
SELECT hu.hint_id
FROM hint_unlocks hu
JOIN hints h ON h.id = hu.hint_id
WHERE hu.team_id = $1 AND h.challenge_id = $2 AND hu.banned_team_id IS NULL;

-- name: DeleteHintUnlocksByTeamID :exec
DELETE FROM hint_unlocks WHERE team_id = $1;

-- name: SoftBanHintUnlocksByTeamID :exec
UPDATE hint_unlocks SET banned_team_id = team_id WHERE team_id = $1 AND banned_team_id IS NULL;

-- name: RestoreHintUnlocksByBannedTeamID :exec
UPDATE hint_unlocks SET banned_team_id = NULL WHERE banned_team_id = $1;
