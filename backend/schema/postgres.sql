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

-- Users (before teams due to captain_id FK)
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
    bracket_id uuid,
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

-- Tags (for challenge categorization)
CREATE TABLE tags (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    color VARCHAR(7) DEFAULT '#6b7280'
);

CREATE TABLE challenge_tags (
    challenge_id uuid NOT NULL,
    tag_id uuid NOT NULL,
    PRIMARY KEY (challenge_id, tag_id)
);

-- Submissions (flag submission attempts history)
CREATE TABLE submissions (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL,
    team_id uuid,
    challenge_id uuid NOT NULL,
    submitted_flag TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT FALSE,
    ip VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Notifications (global announcements and user-specific)
CREATE TABLE notifications (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    type VARCHAR(20) DEFAULT 'info' CHECK (type IN ('info', 'warning', 'success', 'error')),
    is_pinned BOOLEAN DEFAULT FALSE,
    is_global BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_notifications (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL,
    notification_id uuid,
    title VARCHAR(200),
    content TEXT,
    type VARCHAR(20) DEFAULT 'info',
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_notification CHECK (notification_id IS NOT NULL OR (title IS NOT NULL AND content IS NOT NULL))
);

-- Pages (static content)
CREATE TABLE pages (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(200) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    content TEXT NOT NULL,
    is_draft BOOLEAN DEFAULT TRUE,
    order_index INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Brackets (team categories)
CREATE TABLE brackets (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Custom fields (user/team registration)
CREATE TABLE fields (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    field_type VARCHAR(20) NOT NULL CHECK (field_type IN ('text', 'number', 'select', 'boolean')),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('user', 'team')),
    required BOOLEAN DEFAULT FALSE,
    options JSONB,
    order_index INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE field_values (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    field_id uuid NOT NULL,
    entity_id uuid NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (field_id, entity_id)
);

CREATE TABLE api_tokens (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    description VARCHAR(255),
    expires_at TIMESTAMP,
    last_used_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Comments (challenge discussion after CTF ends)
CREATE TABLE comments (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL,
    challenge_id uuid NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Dynamic configs (key-value store, in-memory cached)
CREATE TABLE configs (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    value_type VARCHAR(20) NOT NULL DEFAULT 'string' CHECK (value_type IN ('string', 'int', 'bool', 'json')),
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Ratings (CTF events and global team ratings)
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

-- Indexes
CREATE INDEX idx_solves_user ON solves (user_id);
CREATE INDEX idx_solves_challenge_date ON solves (challenge_id, solved_at);
CREATE UNIQUE INDEX solves_team_challenge_idx ON solves (team_id, challenge_id);
CREATE INDEX idx_users_team ON users (team_id);
CREATE INDEX idx_teams_invite ON teams (invite_token);
CREATE INDEX idx_teams_bracket_id ON teams (bracket_id);
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
CREATE INDEX idx_challenge_tags_tag_id ON challenge_tags (tag_id);
CREATE INDEX idx_challenge_tags_challenge_id ON challenge_tags (challenge_id);
CREATE INDEX idx_submissions_user_id ON submissions (user_id);
CREATE INDEX idx_submissions_team_id ON submissions (team_id);
CREATE INDEX idx_submissions_challenge_id ON submissions (challenge_id);
CREATE INDEX idx_submissions_created_at ON submissions (created_at DESC);
CREATE INDEX idx_submissions_is_correct ON submissions (is_correct);
CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);
CREATE INDEX idx_notifications_is_pinned ON notifications (is_pinned);
CREATE INDEX idx_user_notifications_user_id ON user_notifications (user_id);
CREATE INDEX idx_user_notifications_is_read ON user_notifications (is_read);
CREATE INDEX idx_pages_slug ON pages (slug);
CREATE INDEX idx_pages_is_draft ON pages (is_draft);
CREATE INDEX idx_pages_order ON pages (order_index);
CREATE INDEX idx_field_values_entity ON field_values (entity_id);
CREATE INDEX idx_fields_entity_type ON fields (entity_type);
CREATE INDEX idx_api_tokens_user_id ON api_tokens (user_id);
CREATE INDEX idx_api_tokens_token_hash ON api_tokens (token_hash);
CREATE INDEX idx_comments_challenge_id ON comments (challenge_id);
CREATE INDEX idx_comments_user_id ON comments (user_id);
CREATE INDEX idx_comments_created_at ON comments (created_at);
CREATE INDEX idx_configs_updated_at ON configs (updated_at);
CREATE INDEX idx_team_ratings_team_id ON team_ratings (team_id);
CREATE INDEX idx_team_ratings_ctf_event_id ON team_ratings (ctf_event_id);
CREATE INDEX idx_global_ratings_total_points ON global_ratings (total_points DESC);

-- Foreign keys
ALTER TABLE teams ADD CONSTRAINT fk_teams_captain FOREIGN KEY (captain_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE users ADD CONSTRAINT fk_users_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE SET NULL;
ALTER TABLE solves ADD CONSTRAINT fk_solves_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE solves ADD CONSTRAINT fk_solves_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE solves ADD CONSTRAINT fk_solves_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;
ALTER TABLE verification_tokens ADD CONSTRAINT fk_verification_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE hints ADD CONSTRAINT fk_hints_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;
ALTER TABLE hint_unlocks ADD CONSTRAINT fk_hint_unlocks_hint FOREIGN KEY (hint_id) REFERENCES hints (id) ON DELETE CASCADE;
ALTER TABLE hint_unlocks ADD CONSTRAINT fk_hint_unlocks_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE awards ADD CONSTRAINT fk_awards_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE awards ADD CONSTRAINT fk_awards_created_by FOREIGN KEY (created_by) REFERENCES users (id) ON DELETE SET NULL;
ALTER TABLE files ADD CONSTRAINT fk_files_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;
ALTER TABLE team_audit_log ADD CONSTRAINT fk_team_audit_log_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE team_audit_log ADD CONSTRAINT fk_team_audit_log_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE audit_logs ADD CONSTRAINT fk_audit_logs_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL;
ALTER TABLE submissions ADD CONSTRAINT fk_submissions_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE submissions ADD CONSTRAINT fk_submissions_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE submissions ADD CONSTRAINT fk_submissions_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;
ALTER TABLE challenge_tags ADD CONSTRAINT fk_challenge_tags_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;
ALTER TABLE challenge_tags ADD CONSTRAINT fk_challenge_tags_tag FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE CASCADE;
ALTER TABLE user_notifications ADD CONSTRAINT fk_user_notifications_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE user_notifications ADD CONSTRAINT fk_user_notifications_notification FOREIGN KEY (notification_id) REFERENCES notifications (id) ON DELETE CASCADE;
ALTER TABLE teams ADD CONSTRAINT fk_teams_bracket FOREIGN KEY (bracket_id) REFERENCES brackets (id) ON DELETE SET NULL;
ALTER TABLE field_values ADD CONSTRAINT fk_field_values_field FOREIGN KEY (field_id) REFERENCES fields (id) ON DELETE CASCADE;
ALTER TABLE api_tokens ADD CONSTRAINT fk_api_tokens_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE comments ADD CONSTRAINT fk_comments_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE comments ADD CONSTRAINT fk_comments_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;

-- Singleton rows (required for application)
INSERT INTO competition (id, name) VALUES (1, 'CTF Competition');
INSERT INTO app_settings (id) VALUES (1);
