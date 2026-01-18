CREATE INDEX idx_solves_user ON solves (user_id);
CREATE INDEX idx_solves_challenge_date ON solves (challenge_id, solved_at);
CREATE INDEX idx_users_team ON users (team_id);
CREATE INDEX idx_teams_invite ON teams (invite_token);
CREATE INDEX idx_hints_challenge ON hints (challenge_id);
CREATE INDEX idx_hint_unlocks_hint ON hint_unlocks (hint_id);
CREATE INDEX idx_awards_team ON awards (team_id);
