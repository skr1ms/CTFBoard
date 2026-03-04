-- name: CreatePage :one
INSERT INTO pages (id, title, slug, content, is_draft, order_index, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
RETURNING id, title, slug, content, is_draft, order_index, created_at, updated_at;

-- name: GetPageByID :one
SELECT id, title, slug, content, is_draft, order_index, created_at, updated_at
FROM pages WHERE id = $1;

-- name: GetPageBySlug :one
SELECT id, title, slug, content, is_draft, order_index, created_at, updated_at
FROM pages WHERE slug = $1;

-- name: GetPublishedPages :many
SELECT id, title, slug, order_index
FROM pages WHERE is_draft = FALSE ORDER BY order_index ASC, created_at ASC;

-- name: GetAllPages :many
SELECT id, title, slug, content, is_draft, order_index, created_at, updated_at
FROM pages ORDER BY order_index ASC, created_at ASC;

-- name: UpdatePage :exec
UPDATE pages SET title = $2, slug = $3, content = $4, is_draft = $5, order_index = $6, updated_at = NOW()
WHERE id = $1;

-- name: DeletePage :exec
DELETE FROM pages WHERE id = $1;
