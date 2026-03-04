-- +goose NO TRANSACTION
-- +goose Up
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_solves_team_scoreboard ON solves (team_id) INCLUDE (points_at_solve, solved_at);
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_awards_team_scoreboard ON awards (team_id) INCLUDE (value, created_at);

-- +goose Down
DROP INDEX CONCURRENTLY IF EXISTS idx_solves_team_scoreboard;
DROP INDEX CONCURRENTLY IF EXISTS idx_awards_team_scoreboard;
