CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE TABLE challenges (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    category VARCHAR(50),
    points INT DEFAULT 0,
    flag_hash VARCHAR(255) NOT NULL,
    is_hidden BOOLEAN DEFAULT FALSE
);
CREATE TABLE teams (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    invite_token UUID DEFAULT uuid_generate_v4() NOT NULL,
    captain_id UUID NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    team_id UUID DEFAULT NULL,
    username VARCHAR(50) NOT NULL UNIQUE,
    email VARCHAR(50) NOT NULL UNIQUE,
    role VARCHAR(20) DEFAULT 'user',
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
CREATE TABLE solves (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,
    team_id UUID NOT NULL,
    challenge_id UUID NOT NULL,
    solved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_team_solve UNIQUE (team_id, challenge_id)
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