CREATE TABLE challenges (
    id CHAR(36) PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50),
    points INT DEFAULT 0,
    flag_hash VARCHAR(255) NOT NULL,
    is_hidden TINYINT(1) DEFAULT 0,
    initial_value INT DEFAULT 0,
    min_value INT DEFAULT 0,
    decay INT DEFAULT 0,
    solve_count INT DEFAULT 0
);

CREATE TABLE teams (
    id CHAR(36) PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    invite_token VARCHAR(32) NOT NULL,
    captain_id CHAR(36) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id CHAR(36) PRIMARY KEY,
    team_id CHAR(36) DEFAULT NULL,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(50) NOT NULL UNIQUE,
    role VARCHAR(20) DEFAULT 'user',
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE solves (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    team_id CHAR(36) NOT NULL,
    challenge_id CHAR(36) NOT NULL,
    solved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY unique_team_solve (team_id, challenge_id)
);

CREATE TABLE competition (
    id INT PRIMARY KEY DEFAULT 1,
    name VARCHAR(100) NOT NULL DEFAULT 'CTF Competition',
    start_time TIMESTAMP NULL,
    end_time TIMESTAMP NULL,
    freeze_time TIMESTAMP NULL,
    is_paused TINYINT(1) DEFAULT 0,
    is_public TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT single_row CHECK (id = 1)
);

CREATE TABLE hints (
    id CHAR(36) PRIMARY KEY,
    challenge_id CHAR(36) NOT NULL,
    content TEXT NOT NULL,
    cost INT NOT NULL DEFAULT 0,
    order_index INT NOT NULL DEFAULT 0
);

CREATE TABLE hint_unlocks (
    id CHAR(36) PRIMARY KEY,
    hint_id CHAR(36) NOT NULL,
    team_id CHAR(36) NOT NULL,
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY unique_team_hint (team_id, hint_id)
);

CREATE TABLE awards (
    id CHAR(36) PRIMARY KEY,
    team_id CHAR(36) NOT NULL,
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