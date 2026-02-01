-- Consolidated schema (equivalent to migrations 000001–000021).
-- Use for reference or to create a fresh database from scratch.

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Competition (singleton row, id = 1)
CREATE TABLE competition (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    name VARCHAR(100) NOT NULL DEFAULT 'CTF Competition',
    start_time TIMESTAMP NULL,
    end_time TIMESTAMP NULL,
    freeze_time TIMESTAMP NULL,
    is_paused BOOLEAN DEFAULT FALSE,
    is_public BOOLEAN DEFAULT TRUE,
    flag_regex TEXT,
    mode VARCHAR(20) DEFAULT 'flexible',
    allow_team_switch BOOLEAN DEFAULT TRUE,
    min_team_size INT DEFAULT 1,
    max_team_size INT DEFAULT 10,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Users (before teams due to captain_id FK; team_id FK added later)
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id uuid DEFAULT NULL,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(50) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user',
    is_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Teams
CREATE TABLE teams (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    invite_token uuid DEFAULT uuid_generate_v4() NOT NULL,
    captain_id uuid NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP DEFAULT NULL,
    is_solo BOOLEAN DEFAULT FALSE,
    is_auto_created BOOLEAN DEFAULT FALSE,
    is_banned BOOLEAN DEFAULT FALSE,
    banned_at TIMESTAMP,
    banned_reason TEXT,
    is_hidden BOOLEAN DEFAULT FALSE
);

-- Challenges (with dynamic scoring and optional regex flag)
CREATE TABLE challenges (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
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
    flag_regex TEXT,
    flag_format_regex TEXT
);

-- Solves (one per team per challenge)
CREATE TABLE solves (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL,
    team_id uuid NOT NULL,
    challenge_id uuid NOT NULL,
    solved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_team_solve UNIQUE (team_id, challenge_id)
);

-- Hints (per challenge, cost in points)
CREATE TABLE hints (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    challenge_id uuid NOT NULL,
    content TEXT NOT NULL,
    cost INT NOT NULL DEFAULT 0,
    order_index INT NOT NULL DEFAULT 0
);

-- Hint unlocks (per team per hint)
CREATE TABLE hint_unlocks (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    hint_id uuid NOT NULL,
    team_id uuid NOT NULL,
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_team_hint UNIQUE (team_id, hint_id)
);

-- Awards (bonus/penalty points per team)
CREATE TABLE awards (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id uuid NOT NULL,
    value INT NOT NULL,
    description VARCHAR(255) NOT NULL,
    created_by uuid NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Verification tokens (email verification, password reset)
CREATE TABLE verification_tokens (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('email_verification', 'password_reset')),
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Files (challenge attachments, writeups; stored externally, path in location)
CREATE TABLE files (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('challenge', 'writeup')),
    challenge_id uuid NOT NULL,
    location VARCHAR(512) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Team audit log (join/leave/kick etc. per team)
CREATE TABLE team_audit_log (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id uuid NOT NULL,
    user_id uuid NOT NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Audit logs (global admin/entity actions)
CREATE TABLE audit_logs (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NULL,
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(50),
    ip VARCHAR(45),
    details JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- App settings (singleton, id = 1)
CREATE TABLE app_settings (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    app_name VARCHAR(100) NOT NULL DEFAULT 'CTFBoard',
    verify_emails BOOLEAN NOT NULL DEFAULT TRUE,
    frontend_url VARCHAR(512) NOT NULL DEFAULT 'http://localhost:3000',
    cors_origins TEXT NOT NULL DEFAULT 'http://localhost:3000,http://localhost:5173',
    resend_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    resend_from_email VARCHAR(255) NOT NULL DEFAULT 'noreply@ctfboard.local',
    resend_from_name VARCHAR(100) NOT NULL DEFAULT 'CTFBoard',
    verify_ttl_hours INT NOT NULL DEFAULT 24,
    reset_ttl_hours INT NOT NULL DEFAULT 1,
    submit_limit_per_user INT NOT NULL DEFAULT 10,
    submit_limit_duration_min INT NOT NULL DEFAULT 1,
    scoreboard_visible VARCHAR(20) NOT NULL DEFAULT 'public' CHECK (scoreboard_visible IN ('public', 'hidden', 'admins_only')),
    registration_open BOOLEAN NOT NULL DEFAULT TRUE,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes
CREATE INDEX idx_solves_user ON solves (user_id);
CREATE INDEX idx_solves_challenge_date ON solves (challenge_id, solved_at);
CREATE UNIQUE INDEX solves_team_challenge_idx ON solves (team_id, challenge_id);

CREATE INDEX idx_users_team ON users (team_id);

CREATE INDEX idx_teams_invite ON teams (invite_token);
CREATE INDEX idx_teams_active ON teams (id) WHERE deleted_at IS NULL;
CREATE INDEX idx_teams_is_solo ON teams (is_solo);
CREATE INDEX idx_teams_is_banned ON teams (is_banned);
CREATE INDEX idx_teams_is_hidden ON teams (is_hidden);

CREATE INDEX idx_hints_challenge ON hints (challenge_id);
CREATE INDEX idx_hint_unlocks_hint ON hint_unlocks (hint_id);

CREATE INDEX idx_awards_team ON awards (team_id);

CREATE INDEX idx_verification_token ON verification_tokens (token);
CREATE INDEX idx_verification_user_type ON verification_tokens (user_id, type);
CREATE INDEX idx_verification_expires ON verification_tokens (expires_at);

CREATE INDEX idx_files_challenge_id ON files (challenge_id);
CREATE INDEX idx_files_type ON files (type);

CREATE INDEX idx_team_audit_log_team_id ON team_audit_log (team_id);
CREATE INDEX idx_team_audit_log_user_id ON team_audit_log (user_id);
CREATE INDEX idx_team_audit_log_action ON team_audit_log (action);

CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_entity_type ON audit_logs (entity_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);

-- Foreign keys
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

ALTER TABLE verification_tokens
    ADD CONSTRAINT fk_verification_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE hints
    ADD CONSTRAINT fk_hints_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;

ALTER TABLE hint_unlocks
    ADD CONSTRAINT fk_hint_unlocks_hint FOREIGN KEY (hint_id) REFERENCES hints (id) ON DELETE CASCADE;
ALTER TABLE hint_unlocks
    ADD CONSTRAINT fk_hint_unlocks_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;

ALTER TABLE awards
    ADD CONSTRAINT fk_awards_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE awards
    ADD CONSTRAINT fk_awards_created_by FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL;

ALTER TABLE files
    ADD CONSTRAINT fk_files_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;

ALTER TABLE team_audit_log
    ADD CONSTRAINT fk_team_audit_log_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE team_audit_log
    ADD CONSTRAINT fk_team_audit_log_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;

ALTER TABLE audit_logs
    ADD CONSTRAINT fk_audit_logs_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL;

-- Singleton rows (required for application)
INSERT INTO competition (id, name) VALUES (1, 'CTF Competition');
INSERT INTO app_settings (id) VALUES (1);
