ALTER TABLE teams ADD COLUMN is_solo BOOLEAN DEFAULT FALSE;
ALTER TABLE teams ADD COLUMN is_auto_created BOOLEAN DEFAULT FALSE;
CREATE INDEX idx_teams_is_solo ON teams(is_solo);
