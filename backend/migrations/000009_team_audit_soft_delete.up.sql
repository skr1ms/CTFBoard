CREATE TABLE team_audit_log (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4 (),
    team_id uuid NOT NULL,
    user_id uuid NOT NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_team_audit_log_team_id ON team_audit_log (team_id);

CREATE INDEX idx_team_audit_log_user_id ON team_audit_log (user_id);

CREATE INDEX idx_team_audit_log_action ON team_audit_log (action);

ALTER TABLE teams ADD COLUMN deleted_at TIMESTAMP DEFAULT NULL;

CREATE INDEX idx_teams_active ON teams (id) WHERE deleted_at IS NULL;