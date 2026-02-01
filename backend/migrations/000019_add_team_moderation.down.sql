DROP INDEX IF EXISTS idx_teams_is_hidden;
DROP INDEX IF EXISTS idx_teams_is_banned;
ALTER TABLE teams DROP COLUMN IF EXISTS is_hidden;
ALTER TABLE teams DROP COLUMN IF EXISTS banned_reason;
ALTER TABLE teams DROP COLUMN IF EXISTS banned_at;
ALTER TABLE teams DROP COLUMN IF EXISTS is_banned;
