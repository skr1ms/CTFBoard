DROP INDEX IF EXISTS idx_teams_bracket_id;
ALTER TABLE teams DROP COLUMN IF EXISTS bracket_id;
DROP TABLE IF EXISTS brackets;
