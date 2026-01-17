CREATE TABLE hints (
    id CHAR(36) PRIMARY KEY,
    challenge_id CHAR(36) NOT NULL,
    content TEXT NOT NULL,
    cost INT NOT NULL DEFAULT 0,
    order_index INT NOT NULL DEFAULT 0,
    CONSTRAINT fk_hints_challenge FOREIGN KEY (challenge_id) 
        REFERENCES challenges (id) ON DELETE CASCADE
);

CREATE TABLE hint_unlocks (
    id CHAR(36) PRIMARY KEY,
    hint_id CHAR(36) NOT NULL,
    team_id CHAR(36) NOT NULL,
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY unique_team_hint (team_id, hint_id),
    CONSTRAINT fk_hint_unlocks_hint FOREIGN KEY (hint_id) 
        REFERENCES hints (id) ON DELETE CASCADE,
    CONSTRAINT fk_hint_unlocks_team FOREIGN KEY (team_id) 
        REFERENCES teams (id) ON DELETE CASCADE
);

CREATE TABLE awards (
    id CHAR(36) PRIMARY KEY,
    team_id CHAR(36) NOT NULL,
    value INT NOT NULL,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_awards_team FOREIGN KEY (team_id) 
        REFERENCES teams (id) ON DELETE CASCADE
);
