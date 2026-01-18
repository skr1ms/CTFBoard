CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE challenges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50),
    points INT DEFAULT 0,
    flag_hash VARCHAR(255) NOT NULL,
    is_hidden BOOLEAN DEFAULT FALSE,
    initial_value INT NOT NULL DEFAULT 500,
    min_value INT NOT NULL DEFAULT 100,
    decay INT NOT NULL DEFAULT 20,
    solve_count INT NOT NULL DEFAULT 0
);

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    name VARCHAR(50) NOT NULL UNIQUE,
    invite_token UUID DEFAULT uuid_generate_v4 () NOT NULL,
    captain_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    team_id UUID DEFAULT NULL,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(50) NOT NULL UNIQUE,
    role VARCHAR(20) DEFAULT 'user',
    password_hash VARCHAR(255) NOT NULL,
    is_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE solves (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    user_id UUID NOT NULL,
    team_id UUID NOT NULL,
    challenge_id UUID NOT NULL,
    solved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_team_solve UNIQUE (team_id, challenge_id)
);

CREATE TABLE verification_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    user_id UUID NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,
    type VARCHAR(20) NOT NULL CHECK (
        type IN (
            'email_verification',
            'password_reset'
        )
    ),
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_verification_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);

CREATE TABLE competition (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    name VARCHAR(100) NOT NULL DEFAULT 'CTF Competition',
    start_time TIMESTAMP NULL,
    end_time TIMESTAMP NULL,
    freeze_time TIMESTAMP NULL,
    is_paused BOOLEAN DEFAULT FALSE,
    is_public BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE hints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    challenge_id UUID NOT NULL,
    content TEXT NOT NULL,
    cost INT NOT NULL DEFAULT 0,
    order_index INT NOT NULL DEFAULT 0
);

CREATE TABLE hint_unlocks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    hint_id UUID NOT NULL,
    team_id UUID NOT NULL,
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_team_hint UNIQUE (team_id, hint_id)
);

CREATE TABLE awards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    team_id UUID NOT NULL,
    value INT NOT NULL,
    description VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

ALTER TABLE teams
ADD CONSTRAINT fk_teams_captain FOREIGN KEY (captain_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE users
ADD CONSTRAINT fk_users_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE SET NULL;

ALTER TABLE solves
ADD CONSTRAINT fk_solves_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE solves
ADD CONSTRAINT fk_solves_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;

ALTER TABLE solves
ADD CONSTRAINT fk_solves_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;

ALTER TABLE hints
ADD CONSTRAINT fk_hints_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;

ALTER TABLE hint_unlocks
ADD CONSTRAINT fk_hint_unlocks_hint FOREIGN KEY (hint_id) REFERENCES hints (id) ON DELETE CASCADE;

ALTER TABLE hint_unlocks
ADD CONSTRAINT fk_hint_unlocks_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;

ALTER TABLE awards
ADD CONSTRAINT fk_awards_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;

CREATE INDEX idx_solves_user ON solves (user_id);

CREATE INDEX idx_solves_challenge_date ON solves (challenge_id, solved_at);

CREATE INDEX idx_users_team ON users (team_id);

CREATE INDEX idx_teams_invite ON teams (invite_token);

CREATE INDEX idx_hints_challenge ON hints (challenge_id);

CREATE INDEX idx_hint_unlocks_hint ON hint_unlocks (hint_id);

CREATE INDEX idx_awards_team ON awards (team_id);

CREATE INDEX idx_verification_token ON verification_tokens (token);

CREATE INDEX idx_verification_user_type ON verification_tokens (user_id, type);
