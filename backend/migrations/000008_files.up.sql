CREATE TABLE files (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('challenge', 'writeup')),
    challenge_id uuid NOT NULL,
    location VARCHAR(512) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_files_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE
);

CREATE INDEX idx_files_challenge_id ON files (challenge_id);
CREATE INDEX idx_files_type ON files (type);