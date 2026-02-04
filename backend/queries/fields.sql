-- name: CreateField :exec
INSERT INTO fields (id, name, field_type, entity_type, required, options, order_index, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetFieldByID :one
SELECT id, name, field_type, entity_type, required, options, order_index, created_at
FROM fields WHERE id = $1;

-- name: GetFieldsByEntityType :many
SELECT id, name, field_type, entity_type, required, options, order_index, created_at
FROM fields WHERE entity_type = $1 ORDER BY order_index, name;

-- name: GetAllFields :many
SELECT id, name, field_type, entity_type, required, options, order_index, created_at
FROM fields ORDER BY entity_type, order_index, name;

-- name: UpdateField :exec
UPDATE fields SET name = $2, field_type = $3, required = $4, options = $5, order_index = $6
WHERE id = $1;

-- name: DeleteField :exec
DELETE FROM fields WHERE id = $1;

-- name: GetFieldValuesByEntityID :many
SELECT id, field_id, entity_id, value, created_at
FROM field_values WHERE entity_id = $1;

-- name: DeleteFieldValuesByEntityID :exec
DELETE FROM field_values WHERE entity_id = $1;

-- name: UpsertFieldValue :exec
INSERT INTO field_values (id, field_id, entity_id, value, created_at)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (field_id, entity_id) DO UPDATE SET value = EXCLUDED.value;
