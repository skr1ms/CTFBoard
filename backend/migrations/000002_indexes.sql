-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_solves_team_scoreboard ON solves (team_id) INCLUDE (points_at_solve, solved_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_awards_team_scoreboard ON awards (team_id) INCLUDE (value, created_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_submissions_ip_fail ON submissions(ip, created_at) WHERE is_correct = FALSE;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_teams_scoreboard_filter ON teams (is_banned, is_hidden) WHERE deleted_at IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_submissions_challenge_created_at ON submissions (challenge_id, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_submissions_user_created_at ON submissions (user_id, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_submissions_team_created_at ON submissions (team_id, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_teams_name_trgm ON teams USING gin (name gin_trgm_ops);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_users_was_in_banned_team ON users (was_in_banned_team) WHERE was_in_banned_team = true;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_submissions_banned_user_id ON submissions (banned_user_id) WHERE banned_user_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_submissions_team_challenge_window ON submissions (team_id, challenge_id, created_at DESC) WHERE banned_team_id IS NULL AND banned_user_id IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_solves_challenge_active ON solves (challenge_id, solved_at) WHERE banned_team_id IS NULL AND banned_user_id IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_notifications_global_pinned_created ON notifications (is_global, is_pinned, created_at DESC) WHERE is_global = TRUE;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_user_notifications_unread ON user_notifications (user_id) WHERE is_read = FALSE;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_solves_user_solved_at ON solves (user_id, solved_at DESC) WHERE banned_team_id IS NULL AND banned_user_id IS NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_challenge_opens_team_challenge_opened ON challenge_opens (team_id, challenge_id, opened_at DESC) WHERE team_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_challenge_opens_challenge_opened ON challenge_opens (challenge_id, opened_at DESC);
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS idx_challenge_opens_user_challenge_unique ON challenge_opens (user_id, challenge_id);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_backup_import_jobs_created_at ON backup_import_jobs (created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_backup_import_jobs_status_created_at ON backup_import_jobs (status, created_at DESC);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_challenges_next_challenge_id ON challenges (next_challenge_id) WHERE next_challenge_id IS NOT NULL;
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_challenge_topics_topic_id ON challenge_topics (topic_id);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_challenge_topics_topic_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_challenges_next_challenge_id;
DROP INDEX CONCURRENTLY IF EXISTS idx_backup_import_jobs_status_created_at;
DROP INDEX CONCURRENTLY IF EXISTS idx_backup_import_jobs_created_at;
DROP INDEX CONCURRENTLY IF EXISTS idx_challenge_opens_user_challenge_unique;
DROP INDEX CONCURRENTLY IF EXISTS idx_challenge_opens_challenge_opened;
DROP INDEX CONCURRENTLY IF EXISTS idx_challenge_opens_team_challenge_opened;
DROP INDEX CONCURRENTLY IF EXISTS idx_solves_user_solved_at;
DROP INDEX CONCURRENTLY IF EXISTS idx_user_notifications_unread;
DROP INDEX CONCURRENTLY IF EXISTS idx_notifications_global_pinned_created;
DROP INDEX CONCURRENTLY IF EXISTS idx_solves_challenge_active;
DROP INDEX CONCURRENTLY IF EXISTS idx_submissions_team_challenge_window;
DROP INDEX CONCURRENTLY IF EXISTS idx_solves_team_scoreboard;
DROP INDEX CONCURRENTLY IF EXISTS idx_awards_team_scoreboard;
DROP INDEX CONCURRENTLY IF EXISTS idx_submissions_ip_fail;
DROP INDEX CONCURRENTLY IF EXISTS idx_teams_scoreboard_filter;
DROP INDEX CONCURRENTLY IF EXISTS idx_submissions_challenge_created_at;
DROP INDEX CONCURRENTLY IF EXISTS idx_submissions_user_created_at;
DROP INDEX CONCURRENTLY IF EXISTS idx_submissions_team_created_at;
DROP INDEX CONCURRENTLY IF EXISTS idx_teams_name_trgm;
DROP INDEX CONCURRENTLY IF EXISTS idx_users_was_in_banned_team;
DROP INDEX CONCURRENTLY IF EXISTS idx_submissions_banned_user_id;
