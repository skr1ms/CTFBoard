-- name: GetAllConfigs :many
SELECT key, value, value_type, description, updated_at
FROM configs ORDER BY key ASC;

-- name: GetConfigByKey :one
SELECT key, value, value_type, description, updated_at
FROM configs WHERE key = $1;

-- name: UpsertConfig :exec
INSERT INTO configs (key, value, value_type, description, updated_at)
VALUES ($1, $2, $3, $4, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO UPDATE SET
    value = EXCLUDED.value,
    value_type = EXCLUDED.value_type,
    description = EXCLUDED.description,
    updated_at = CURRENT_TIMESTAMP;

-- name: DeleteConfig :exec
DELETE FROM configs WHERE key = $1;
