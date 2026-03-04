-- name: CreateHintUnlock :exec
INSERT INTO hint_unlocks (id, hint_id, team_id)
VALUES ($1, $2, $3)
ON CONFLICT (team_id, hint_id) DO NOTHING;

-- name: GetHintUnlockByTeamAndHint :one
SELECT id, hint_id, team_id, unlocked_at
FROM hint_unlocks
WHERE team_id = $1 AND hint_id = $2;

-- name: GetHintUnlockByTeamAndHintForUpdate :one
SELECT id, hint_id, team_id, unlocked_at
FROM hint_unlocks
WHERE team_id = $1 AND hint_id = $2
FOR UPDATE;

-- name: GetAllHintUnlocksSimple :many
SELECT id, hint_id, team_id, unlocked_at
FROM hint_unlocks
ORDER BY unlocked_at;

-- name: GetAllHintUnlocks :many
SELECT hu.id, hu.hint_id, hu.team_id, hu.unlocked_at,
    h.challenge_id, h.cost AS hint_cost
FROM hint_unlocks hu
JOIN hints h ON h.id = hu.hint_id
ORDER BY hu.unlocked_at DESC
LIMIT $1 OFFSET $2;

-- name: CountAllHintUnlocks :one
SELECT COUNT(*)::int FROM hint_unlocks;

-- name: CountHintUnlocksByTeamID :one
SELECT COUNT(*)::int FROM hint_unlocks WHERE team_id = $1;

-- name: GetUnlockedHintIDs :many
SELECT hu.hint_id
FROM hint_unlocks hu
JOIN hints h ON h.id = hu.hint_id
WHERE hu.team_id = $1 AND h.challenge_id = $2;
