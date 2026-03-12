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

-- +goose Down
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
