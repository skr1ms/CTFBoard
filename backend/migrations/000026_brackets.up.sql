CREATE TABLE brackets (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE teams ADD COLUMN bracket_id uuid REFERENCES brackets(id) ON DELETE SET NULL;
CREATE INDEX idx_teams_bracket_id ON teams (bracket_id);
