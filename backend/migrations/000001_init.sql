-- +goose Up
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =============================================================================
-- Singletons
-- =============================================================================

-- Competition (singleton row, id = 1)
CREATE TABLE competition (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    name VARCHAR(100) NOT NULL DEFAULT 'CTF Competition',
    start_time TIMESTAMPTZ NULL,
    end_time TIMESTAMPTZ NULL,
    freeze_time TIMESTAMPTZ NULL,
    is_paused BOOLEAN DEFAULT FALSE,
    paused_at TIMESTAMPTZ NULL,
    is_public BOOLEAN DEFAULT TRUE,
    flag_regex TEXT,
    mode VARCHAR(20) DEFAULT 'flexible' CHECK (mode IN ('solo_only', 'teams_only', 'flexible')),
    allow_team_switch BOOLEAN DEFAULT TRUE,
    min_team_size INT DEFAULT 1 CHECK (min_team_size >= 1),
    max_team_size INT DEFAULT 10 CHECK (max_team_size >= 1),
    CONSTRAINT chk_max_team_size_gte_min CHECK (max_team_size >= min_team_size),
    keep_scoreboard_frozen_after_end BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- App settings (singleton, id = 1)
CREATE TABLE app_settings (
    id INT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    app_name VARCHAR(100) NOT NULL DEFAULT 'AstroCTFb',
    verify_emails BOOLEAN NOT NULL DEFAULT TRUE,
    frontend_url VARCHAR(512) NOT NULL DEFAULT 'http://localhost:3000',
    cors_origins TEXT NOT NULL DEFAULT 'http://localhost:3000,http://localhost:5173',
    resend_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    resend_from_email VARCHAR(255) NOT NULL DEFAULT 'noreply@astroctfb.local',
    resend_from_name VARCHAR(100) NOT NULL DEFAULT 'AstroCTFb',
    verify_ttl_hours INT NOT NULL DEFAULT 24,
    reset_ttl_hours INT NOT NULL DEFAULT 1,
    submit_limit_per_user INT NOT NULL DEFAULT 10,
    submit_limit_duration_min INT NOT NULL DEFAULT 1,
    scoreboard_visible VARCHAR(20) NOT NULL DEFAULT 'public' CHECK (scoreboard_visible IN ('public', 'hidden', 'admins_only')),
    registration_open BOOLEAN NOT NULL DEFAULT TRUE,
    default_per_page INT NOT NULL DEFAULT 50,
    max_per_page INT NOT NULL DEFAULT 100,
    csv_export_max_rows INT NOT NULL DEFAULT 100000,
    rate_limit_login_per_minute INT NOT NULL DEFAULT 10,
    rate_limit_register_per_minute INT NOT NULL DEFAULT 5,
    rate_limit_forgot_password_per_minute INT NOT NULL DEFAULT 3,
    rate_limit_reset_password_per_minute INT NOT NULL DEFAULT 5,
    rate_limit_logout_per_minute INT NOT NULL DEFAULT 10,
    rate_limit_refresh_per_minute INT NOT NULL DEFAULT 10,
    rate_limit_scoreboard_per_minute INT NOT NULL DEFAULT 30,
    rate_limit_general_ip_per_minute INT NOT NULL DEFAULT 100,
    rate_limit_verify_email_per_minute INT NOT NULL DEFAULT 10,
    rate_limit_oauth_callback_per_minute INT NOT NULL DEFAULT 20,
    rate_limit_oauth_redirect_per_minute INT NOT NULL DEFAULT 20,
    rate_limit_comment_per_minute INT NOT NULL DEFAULT 30,
    max_teams INT NOT NULL DEFAULT 0,
    writeup_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    oauth_github_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    oauth_google_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- =============================================================================
-- Users & Teams
-- =============================================================================

-- Brackets (team categories)
CREATE TABLE brackets (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

-- Users (team_id FK deferred - see ALTER TABLE below)
CREATE TABLE users (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id uuid DEFAULT NULL,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(254) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    role VARCHAR(20) DEFAULT 'user' CHECK (role IN ('user', 'admin')),
    is_verified BOOLEAN DEFAULT FALSE,
    verified_at TIMESTAMPTZ NULL,
    is_banned BOOLEAN NOT NULL DEFAULT FALSE,
    banned_at TIMESTAMPTZ NULL,
    banned_reason TEXT NULL,
    was_in_banned_team BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_users_team ON users (team_id);
CREATE INDEX idx_users_is_banned ON users (is_banned);
CREATE INDEX idx_users_username_trgm ON users USING gin (username gin_trgm_ops);

-- Teams (captain_id and bracket_id are inline; users.team_id added below)
CREATE TABLE teams (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL,
    invite_token uuid DEFAULT uuid_generate_v4() NOT NULL,
    captain_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    bracket_id uuid REFERENCES brackets (id) ON DELETE SET NULL,
    is_solo BOOLEAN DEFAULT FALSE,
    is_auto_created BOOLEAN DEFAULT FALSE,
    is_banned BOOLEAN DEFAULT FALSE,
    banned_at TIMESTAMPTZ,
    banned_reason TEXT,
    is_hidden BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ DEFAULT NULL,
    invite_token_expires_at TIMESTAMPTZ DEFAULT NULL
);

CREATE UNIQUE INDEX teams_name_active_key ON teams (name) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX teams_invite_token_active_key ON teams (invite_token) WHERE deleted_at IS NULL;
CREATE INDEX idx_teams_bracket_id ON teams (bracket_id);
CREATE INDEX idx_teams_active ON teams (id) WHERE deleted_at IS NULL;
CREATE INDEX idx_teams_is_solo ON teams (is_solo);
CREATE INDEX idx_teams_is_banned ON teams (is_banned);
CREATE INDEX idx_teams_is_hidden ON teams (is_hidden);

-- Closes the circular dependency: users.team_id -> teams.id
ALTER TABLE users ADD CONSTRAINT fk_users_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE SET NULL;

-- OAuth accounts (external identity provider links)
CREATE TABLE oauth_accounts (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    provider VARCHAR(20) NOT NULL CHECK (provider IN ('github', 'google')),
    provider_user_id VARCHAR(255) NOT NULL,
    access_token TEXT,
    refresh_token TEXT,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, provider_user_id)
);

CREATE INDEX idx_oauth_accounts_user_id ON oauth_accounts (user_id);

-- Verification tokens (email verification, password reset)
CREATE TABLE verification_tokens (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token VARCHAR(64) NOT NULL UNIQUE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('email_verification', 'password_reset')),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_verification_user_type ON verification_tokens (user_id, type);
CREATE INDEX idx_verification_expires ON verification_tokens (expires_at);

-- API tokens (user API access tokens)
CREATE TABLE api_tokens (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    description VARCHAR(255),
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_api_tokens_user_id ON api_tokens (user_id);

-- =============================================================================
-- Challenges
-- =============================================================================

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

-- Tags (for challenge categorization)
CREATE TABLE tags (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    color VARCHAR(7) DEFAULT '#6b7280'
);

-- Challenge–tag many-to-many
CREATE TABLE challenge_tags (
    challenge_id uuid NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    tag_id uuid NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (challenge_id, tag_id)
);

CREATE INDEX idx_challenge_tags_challenge_id ON challenge_tags (challenge_id);
CREATE INDEX idx_challenge_tags_tag_id ON challenge_tags (tag_id);

-- Challenge prerequisites (challenge requires solving other challenges)
CREATE TABLE challenge_requirements (
    challenge_id uuid NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    required_challenge_id uuid NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    PRIMARY KEY (challenge_id, required_challenge_id),
    CONSTRAINT chk_no_self_requirement CHECK (challenge_id != required_challenge_id)
);

CREATE INDEX idx_challenge_requirements_challenge_id ON challenge_requirements (challenge_id);
CREATE INDEX idx_challenge_requirements_required_id ON challenge_requirements (required_challenge_id);

-- Hints (per challenge)
CREATE TABLE hints (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    challenge_id uuid NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    cost INT NOT NULL DEFAULT 0,
    order_index INT NOT NULL DEFAULT 0
);

CREATE INDEX idx_hints_challenge ON hints (challenge_id);

-- Hint unlocks (per team per hint)
CREATE TABLE hint_unlocks (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    hint_id uuid NOT NULL REFERENCES hints (id) ON DELETE CASCADE,
    team_id uuid NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    unlocked_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    banned_team_id uuid NULL REFERENCES teams (id) ON DELETE SET NULL,
    CONSTRAINT unique_team_hint UNIQUE (team_id, hint_id)
);

CREATE INDEX idx_hint_unlocks_hint ON hint_unlocks (hint_id);
CREATE INDEX idx_hint_unlocks_banned_team_id ON hint_unlocks (banned_team_id) WHERE banned_team_id IS NOT NULL;

-- Files (challenge attachments, writeups; stored externally)
CREATE TABLE files (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    type VARCHAR(20) NOT NULL CHECK (type IN ('challenge', 'writeup')),
    challenge_id uuid NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    location VARCHAR(512) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    size BIGINT NOT NULL,
    sha256 VARCHAR(64) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_files_challenge_id ON files (challenge_id);
CREATE INDEX idx_files_type ON files (type);

-- Solutions (one per challenge, stores markdown writeup content)
CREATE TABLE solutions (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    challenge_id uuid UNIQUE NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    content TEXT NOT NULL DEFAULT ''
);


-- =============================================================================
-- Competition activity
-- =============================================================================

-- Solves (one per team per challenge)
CREATE TABLE solves (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    team_id uuid NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    challenge_id uuid NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    solved_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    points_at_solve INT NOT NULL DEFAULT 0,
    banned_team_id uuid NULL REFERENCES teams (id) ON DELETE SET NULL,
    banned_user_id uuid NULL REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT unique_team_solve UNIQUE (team_id, challenge_id)
);

CREATE INDEX idx_solves_user ON solves (user_id);
CREATE INDEX idx_solves_challenge_date ON solves (challenge_id, solved_at);
CREATE INDEX idx_solves_team_challenge ON solves (team_id, challenge_id);
CREATE INDEX idx_solves_banned_team_id ON solves (banned_team_id) WHERE banned_team_id IS NOT NULL;
CREATE INDEX idx_solves_banned_user_id ON solves (banned_user_id) WHERE banned_user_id IS NOT NULL;

-- Submissions (flag submission attempts history)
CREATE TABLE submissions (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    team_id uuid REFERENCES teams (id) ON DELETE CASCADE,
    challenge_id uuid NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    submitted_flag TEXT NOT NULL,
    is_correct BOOLEAN NOT NULL DEFAULT FALSE,
    ip VARCHAR(45),
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    banned_team_id uuid NULL REFERENCES teams (id) ON DELETE SET NULL,
    banned_user_id uuid NULL REFERENCES users (id) ON DELETE SET NULL
);

CREATE INDEX idx_submissions_user_id ON submissions (user_id);
CREATE INDEX idx_submissions_team_id ON submissions (team_id);
CREATE INDEX idx_submissions_challenge_id ON submissions (challenge_id);
CREATE INDEX idx_submissions_created_at ON submissions (created_at DESC);
CREATE INDEX idx_submissions_is_correct ON submissions (is_correct);
CREATE INDEX IF NOT EXISTS idx_submissions_user_correct ON submissions (user_id, is_correct);
CREATE INDEX IF NOT EXISTS idx_submissions_team_correct ON submissions (team_id, is_correct);
CREATE INDEX idx_submissions_banned_team_id ON submissions (banned_team_id) WHERE banned_team_id IS NOT NULL;

-- Comments (challenge discussion after CTF ends)
CREATE TABLE comments (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    challenge_id uuid NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_comments_challenge_id ON comments (challenge_id);
CREATE INDEX idx_comments_user_id ON comments (user_id);
CREATE INDEX idx_comments_created_at ON comments (created_at);

-- =============================================================================
-- Awards & Audit
-- =============================================================================

-- Awards (bonus/penalty points per team)
CREATE TABLE awards (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id uuid NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    value INT NOT NULL,
    description VARCHAR(255) NOT NULL,
    created_by uuid REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    banned_team_id uuid NULL REFERENCES teams (id) ON DELETE SET NULL
);

CREATE INDEX idx_awards_team ON awards (team_id);
CREATE INDEX idx_awards_banned_team_id ON awards (banned_team_id) WHERE banned_team_id IS NOT NULL;

-- Team audit log (join/leave/kick etc. per team)
CREATE TABLE team_audit_log (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id uuid NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    user_id uuid REFERENCES users (id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_team_audit_log_team_id ON team_audit_log (team_id);
CREATE INDEX idx_team_audit_log_user_id ON team_audit_log (user_id);
CREATE INDEX idx_team_audit_log_action ON team_audit_log (action);
CREATE INDEX idx_team_audit_log_team_action ON team_audit_log (team_id, action, created_at DESC);

-- Audit logs (global admin/entity actions)
CREATE TABLE audit_logs (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid REFERENCES users (id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL,
    entity_type VARCHAR(50) NOT NULL,
    entity_id VARCHAR(50),
    ip VARCHAR(45),
    details JSONB,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_audit_logs_user_id ON audit_logs (user_id);
CREATE INDEX idx_audit_logs_entity_type ON audit_logs (entity_type);
CREATE INDEX idx_audit_logs_created_at ON audit_logs (created_at);

-- =============================================================================
-- Content
-- =============================================================================

-- Notifications (global announcements)
CREATE TABLE notifications (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(200) NOT NULL,
    content TEXT NOT NULL,
    type VARCHAR(20) DEFAULT 'info' CHECK (type IN ('info', 'warning', 'success', 'error')),
    is_pinned BOOLEAN DEFAULT FALSE,
    is_global BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_notifications_created_at ON notifications (created_at DESC);
CREATE INDEX idx_notifications_is_pinned ON notifications (is_pinned);

-- User notifications (global announcements and user-specific)
CREATE TABLE user_notifications (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    notification_id uuid REFERENCES notifications (id) ON DELETE CASCADE,
    title VARCHAR(200),
    content TEXT,
    type VARCHAR(20) DEFAULT 'info',
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT chk_notification CHECK (notification_id IS NOT NULL OR (title IS NOT NULL AND content IS NOT NULL))
);

CREATE INDEX idx_user_notifications_user_id ON user_notifications (user_id);
CREATE INDEX idx_user_notifications_is_read ON user_notifications (is_read);

-- Pages (static content)
CREATE TABLE pages (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(200) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    content TEXT NOT NULL,
    is_draft BOOLEAN DEFAULT TRUE,
    order_index INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_pages_is_draft ON pages (is_draft);
CREATE INDEX idx_pages_order ON pages (order_index);

-- =============================================================================
-- Custom fields
-- =============================================================================

-- Custom fields (user/team registration)
CREATE TABLE fields (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL,
    field_type VARCHAR(20) NOT NULL CHECK (field_type IN ('text', 'number', 'select', 'boolean')),
    entity_type VARCHAR(20) NOT NULL CHECK (entity_type IN ('user', 'team')),
    required BOOLEAN DEFAULT FALSE,
    options JSONB,
    order_index INT DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_fields_entity_type ON fields (entity_type);

-- Field values (custom field values for users or teams)
CREATE TABLE field_values (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    field_id uuid NOT NULL REFERENCES fields (id) ON DELETE CASCADE,
    entity_id uuid NOT NULL,
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (field_id, entity_id)
);

CREATE INDEX idx_field_values_entity ON field_values (entity_id);

-- =============================================================================
-- Config & Tracking
-- =============================================================================

-- Dynamic configs (key-value store, in-memory cached)
CREATE TABLE configs (
    key VARCHAR(100) PRIMARY KEY,
    value TEXT NOT NULL,
    value_type VARCHAR(20) NOT NULL DEFAULT 'string' CHECK (value_type IN ('string', 'int', 'bool', 'json')),
    description TEXT,
    updated_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_configs_updated_at ON configs (updated_at);

-- IP/user-agent tracking per user
CREATE TABLE tracking (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    ip VARCHAR(45) NOT NULL,
    user_agent TEXT,
    tracked_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_tracking_user_id ON tracking (user_id);
CREATE INDEX idx_tracking_ip ON tracking (ip);
CREATE INDEX idx_tracking_tracked_at ON tracking (tracked_at DESC);
CREATE INDEX idx_tracking_ip_user_id ON tracking (ip, user_id);

-- Challenge open events (tracks when users view challenges)
CREATE TABLE challenge_opens (
    id uuid PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id uuid NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    challenge_id uuid NOT NULL REFERENCES challenges (id) ON DELETE CASCADE,
    ip VARCHAR(45),
    opened_at TIMESTAMPTZ DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_challenge_opens_user_id ON challenge_opens (user_id);
CREATE INDEX idx_challenge_opens_challenge_id ON challenge_opens (challenge_id);
CREATE INDEX idx_challenge_opens_opened_at ON challenge_opens (opened_at DESC);

-- =============================================================================
-- Seed data
-- =============================================================================

INSERT INTO competition (id, name) VALUES (1, 'CTF Competition');
INSERT INTO app_settings (id) VALUES (1);

-- +goose Down
DROP SCHEMA public CASCADE;
CREATE SCHEMA public;
GRANT ALL ON SCHEMA public TO public;
