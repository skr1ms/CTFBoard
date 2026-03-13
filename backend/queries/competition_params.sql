-- name: GetAllConfigs :many
SELECT key, value, value_type, description, category, updated_at
FROM configs ORDER BY key ASC;

-- name: GetConfigByKey :one
SELECT key, value, value_type, description, category, updated_at
FROM configs WHERE key = $1;

-- name: GetConfigByKeyForUpdate :one
SELECT key, value, value_type, description, category, updated_at
FROM configs WHERE key = $1 FOR UPDATE;

-- name: UpsertConfig :exec
INSERT INTO configs (key, value, value_type, description, category, updated_at)
VALUES ($1, $2, $3, $4, $5, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    value_type = EXCLUDED.value_type,
    description = EXCLUDED.description,
    category = EXCLUDED.category,
    updated_at = CURRENT_TIMESTAMP;

-- name: DeleteConfig :exec
DELETE FROM configs WHERE key = $1;
