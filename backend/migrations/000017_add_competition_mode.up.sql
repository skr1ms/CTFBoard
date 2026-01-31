ALTER TABLE competition ADD COLUMN mode VARCHAR(20) DEFAULT 'flexible';
ALTER TABLE competition ADD COLUMN allow_team_switch BOOLEAN DEFAULT TRUE;
ALTER TABLE competition ADD COLUMN min_team_size INT DEFAULT 1;
ALTER TABLE competition ADD COLUMN max_team_size INT DEFAULT 10;
