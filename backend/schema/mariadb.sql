CREATE TABLE challenges (
    id CHAR(36) PRIMARY KEY,
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50),
    points INT DEFAULT 0,
    flag_hash VARCHAR(255) NOT NULL,
    is_hidden TINYINT(1) DEFAULT 0
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
ALTER TABLE teams
ADD CONSTRAINT fk_teams_captain FOREIGN KEY (captain_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE users
ADD CONSTRAINT fk_users_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE
SET NULL;
ALTER TABLE solves
ADD CONSTRAINT fk_solves_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE;
ALTER TABLE solves
ADD CONSTRAINT fk_solves_team FOREIGN KEY (team_id) REFERENCES teams (id) ON DELETE CASCADE;
ALTER TABLE solves
ADD CONSTRAINT fk_solves_challenge FOREIGN KEY (challenge_id) REFERENCES challenges (id) ON DELETE CASCADE;