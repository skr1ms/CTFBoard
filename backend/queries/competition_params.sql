-- name: GetAllConfigs :many
SELECT key, value, value_type, description, category, updated_at
FROM configs ORDER BY key ASC;

-- name: GetConfigsByCategory :many
SELECT key, value, value_type, description, category, updated_at
FROM configs
WHERE category = $1
ORDER BY key ASC;

-- name: GetConfigByKey :one
SELECT key, value, value_type, description, category, updated_at
FROM configs WHERE key = $1;

-- name: GetConfigByKeyForUpdate :one
SELECT key, value, value_type, description, category, updated_at
FROM configs WHERE key = $1 FOR UPDATE;

-- name: ReconcileCTFNameDefault :exec
UPDATE configs
SET value = $1
WHERE key = 'ctf_name'
  AND value = $2;

-- name: UpsertConfig :exec
INSERT INTO configs (key, value, value_type, description, category, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    value_type = EXCLUDED.value_type,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    updated_at = EXCLUDED.updated_at;

-- name: DeleteConfig :exec
DELETE FROM configs WHERE key = $1;
