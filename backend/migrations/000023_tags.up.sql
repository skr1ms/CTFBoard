CREATE TABLE tags (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    color VARCHAR(7) DEFAULT '#6b7280'
);

CREATE TABLE challenge_tags (
    challenge_id uuid NOT NULL REFERENCES challenges(id) ON DELETE CASCADE,
    tag_id uuid NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (challenge_id, tag_id)
);

CREATE INDEX idx_challenge_tags_tag_id ON challenge_tags (tag_id);
CREATE INDEX idx_challenge_tags_challenge_id ON challenge_tags (challenge_id);
