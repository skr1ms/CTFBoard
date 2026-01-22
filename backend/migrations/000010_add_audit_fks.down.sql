ALTER TABLE team_audit_log
DROP CONSTRAINT IF EXISTS fk_team_audit_log_team;

ALTER TABLE team_audit_log
DROP CONSTRAINT IF EXISTS fk_team_audit_log_user;