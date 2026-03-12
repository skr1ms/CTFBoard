-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_global_pinned_created ON notifications (is_global, is_pinned, created_at DESC) WHERE is_global = TRUE;
-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_notifications_global_pinned_created;
