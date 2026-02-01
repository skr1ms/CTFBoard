ALTER TABLE teams ADD COLUMN is_banned BOOLEAN DEFAULT FALSE;
ALTER TABLE teams ADD COLUMN banned_at TIMESTAMP;
ALTER TABLE teams ADD COLUMN banned_reason TEXT;
ALTER TABLE teams ADD COLUMN is_hidden BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_teams_is_banned ON teams(is_banned);
CREATE INDEX idx_teams_is_hidden ON teams(is_hidden);
