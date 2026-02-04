-- name: CreateNotification :one
INSERT INTO notifications (id, title, content, type, is_pinned, is_global, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, title, content, type, is_pinned, is_global, created_at;

-- name: GetNotificationByID :one
SELECT id, title, content, type, is_pinned, is_global, created_at
FROM notifications WHERE id = $1;

-- name: GetAllNotifications :many
SELECT id, title, content, type, is_pinned, is_global, created_at
FROM notifications
WHERE is_global = TRUE
ORDER BY is_pinned DESC, created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateNotification :exec
UPDATE notifications
SET title = $2, content = $3, type = $4, is_pinned = $5
WHERE id = $1;

-- name: DeleteNotification :exec
DELETE FROM notifications WHERE id = $1;

-- name: CreateUserNotification :one
INSERT INTO user_notifications (id, user_id, notification_id, title, content, type, is_read, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING id, user_id, notification_id, title, content, type, is_read, created_at;

-- name: GetUserNotifications :many
SELECT id, user_id, notification_id, title, content, type, is_read, created_at
FROM user_notifications
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: MarkUserNotificationAsRead :exec
UPDATE user_notifications
SET is_read = TRUE
WHERE id = $1 AND user_id = $2;

-- name: CountUnreadUserNotifications :one
SELECT COUNT(*) FROM user_notifications WHERE user_id = $1 AND is_read = FALSE;

-- name: DeleteUserNotification :exec
DELETE FROM user_notifications WHERE id = $1 AND user_id = $2;
