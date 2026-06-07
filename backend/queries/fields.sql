-- name: CreateField :exec
INSERT INTO fields (id, name, description, field_type, entity_type, required, is_public, editable, options, order_index, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetFieldByID :one
SELECT id, name, description, field_type, entity_type, required, is_public, editable, options, order_index, created_at
FROM fields WHERE id = $1;

-- name: GetFieldsByEntityType :many
SELECT id, name, description, field_type, entity_type, required, is_public, editable, options, order_index, created_at
FROM fields WHERE entity_type = $1 ORDER BY order_index, name;

-- name: GetAllFields :many
SELECT id, name, description, field_type, entity_type, required, is_public, editable, options, order_index, created_at
FROM fields ORDER BY entity_type, order_index, name;

-- name: UpdateField :exec
UPDATE fields SET name = $2, description = $3, field_type = $4, required = $5, is_public = $6, editable = $7, options = $8, order_index = $9
WHERE id = $1;

-- name: DeleteField :exec
DELETE FROM fields WHERE id = $1;

-- name: GetFieldValuesByEntityID :many
SELECT id, field_id, entity_id, value, created_at
FROM field_values WHERE entity_id = $1;

-- name: GetAllFieldValues :many
SELECT id, field_id, entity_id, value, created_at
FROM field_values ORDER BY field_id, entity_id;

-- name: DeleteFieldValuesByEntityID :exec
DELETE FROM field_values WHERE entity_id = $1;
