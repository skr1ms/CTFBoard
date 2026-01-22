ALTER TABLE team_audit_log
ADD CONSTRAINT fk_team_audit_log_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;

ALTER TABLE team_audit_log
ADD CONSTRAINT fk_team_audit_log_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;