DROP INDEX IF EXISTS idx_teams_is_solo;
ALTER TABLE teams DROP COLUMN IF EXISTS is_auto_created;
ALTER TABLE teams DROP COLUMN IF EXISTS is_solo;
