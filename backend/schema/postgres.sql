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
    solve_count INT NOT NULL DEFAULT 0,
    is_regex BOOLEAN DEFAULT FALSE,
    is_case_insensitive BOOLEAN DEFAULT FALSE,
    flag_regex TEXT
);

CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    name VARCHAR(50) NOT NULL UNIQUE,
    invite_token UUID DEFAULT uuid_generate_v4 () NOT NULL,
    captain_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL
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
    solved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
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
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE competition (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    name VARCHAR(100) NOT NULL DEFAULT 'CTF Competition',
    start_time TIMESTAMP NULL,
    end_time TIMESTAMP NULL,
    freeze_time TIMESTAMP NULL,
    is_paused BOOLEAN DEFAULT FALSE,
    is_public BOOLEAN DEFAULT TRUE,
    flag_regex TEXT,
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
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE awards (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    team_id UUID NOT NULL,
    value INT NOT NULL,
    description VARCHAR(255) NOT NULL,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE files (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    type VARCHAR(20) NOT NULL CHECK (
        type IN ('challenge', 'writeup')
    ),
    challenge_id UUID NOT NULL,
    location VARCHAR(512) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE team_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4 (),
    team_id UUID NOT NULL,
    user_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID,
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(50),
    ip VARCHAR(45),
    details JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indices
CREATE INDEX idx_solves_user ON solves (user_id);

CREATE INDEX idx_solves_challenge_date ON solves (challenge_id, solved_at);

CREATE INDEX idx_users_team ON users (team_id);

CREATE INDEX idx_team_audit_log_team_id ON team_audit_log (team_id);

CREATE INDEX idx_team_audit_log_user_id ON team_audit_log (user_id);

CREATE INDEX idx_team_audit_log_action ON team_audit_log (action);

CREATE INDEX idx_teams_invite ON teams (invite_token);

CREATE INDEX idx_teams_active ON teams (id) WHERE deleted_at IS NULL;

CREATE INDEX idx_hints_challenge ON hints (challenge_id);

CREATE INDEX idx_hint_unlocks_hint ON hint_unlocks (hint_id);

CREATE INDEX idx_awards_team ON awards (team_id);

CREATE INDEX idx_verification_token ON verification_tokens (token);

CREATE INDEX idx_verification_user_type ON verification_tokens (user_id, type);

CREATE INDEX idx_verification_expires ON verification_tokens (expires_at);

CREATE INDEX idx_files_challenge_id ON files (challenge_id);

CREATE INDEX idx_files_type ON files (type);

CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);

CREATE INDEX idx_audit_logs_entity_type ON audit_logs (entity_type);

CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);

-- Constraint

-- Teams
ALTER TABLE teams
ADD CONSTRAINT fk_teams_captain FOREIGN KEY (captain_id) REFERENCES users (id) ON DELETE CASCADE;

-- Users
ALTER TABLE users
ADD CONSTRAINT fk_users_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE SET NULL;

-- Solves
ALTER TABLE solves
ADD CONSTRAINT unique_team_solve UNIQUE (team_id, challenge_id);

ALTER TABLE solves
ADD CONSTRAINT fk_solves_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE solves
ADD CONSTRAINT fk_solves_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;

ALTER TABLE solves
ADD CONSTRAINT fk_solves_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;

-- Verification Tokens
ALTER TABLE verification_tokens
ADD CONSTRAINT fk_verification_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

-- Hints
ALTER TABLE hints
ADD CONSTRAINT fk_hints_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;

-- Hint Unlocks
ALTER TABLE hint_unlocks
ADD CONSTRAINT unique_team_hint UNIQUE (team_id, hint_id);

ALTER TABLE hint_unlocks
ADD CONSTRAINT fk_hint_unlocks_hint FOREIGN KEY (hint_id) REFERENCES hints (id) ON DELETE CASCADE;

ALTER TABLE hint_unlocks
ADD CONSTRAINT fk_hint_unlocks_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;

-- Awards
ALTER TABLE awards
ADD CONSTRAINT fk_awards_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;

-- Files
ALTER TABLE files
ADD CONSTRAINT fk_files_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;

-- Team Audit Log
ALTER TABLE team_audit_log
ADD CONSTRAINT fk_team_audit_log_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;

ALTER TABLE team_audit_log
ADD CONSTRAINT fk_team_audit_log_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

-- Audit Logs
ALTER TABLE audit_logs
ADD CONSTRAINT fk_audit_logs_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL;