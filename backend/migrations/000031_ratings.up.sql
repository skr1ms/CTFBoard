CREATE TABLE ctf_events (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(200) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    weight DECIMAL(3,2) DEFAULT 1.0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE team_ratings (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id uuid NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
    ctf_event_id uuid NOT NULL REFERENCES ctf_events(id) ON DELETE CASCADE,
    rank INT NOT NULL,
    score INT NOT NULL,
    rating_points DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (team_id, ctf_event_id)
);

CREATE TABLE global_ratings (
    team_id uuid PRIMARY KEY REFERENCES teams(id) ON DELETE CASCADE,
    total_points DECIMAL(12,2) NOT NULL DEFAULT 0,
    events_count INT NOT NULL DEFAULT 0,
    best_rank INT,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_team_ratings_team_id ON team_ratings (team_id);
CREATE INDEX idx_team_ratings_ctf_event_id ON team_ratings (ctf_event_id);
CREATE INDEX idx_global_ratings_total_points ON global_ratings (total_points DESC);
