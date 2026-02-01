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

INSERT INTO app_settings (id) VALUES (1);
