DROP INDEX IF EXISTS idx_teams_active;

ALTER TABLE teams DROP COLUMN IF EXISTS deleted_at;

DROP INDEX IF EXISTS idx_team_audit_log_action;

DROP INDEX IF EXISTS idx_team_audit_log_user_id;

DROP INDEX IF EXISTS idx_team_audit_log_team_id;

DROP TABLE IF EXISTS team_audit_log;