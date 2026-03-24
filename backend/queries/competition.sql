-- name: GetCompetition :one
SELECT id, name, start_time, end_time, freeze_time, is_paused, paused_at, is_public,
       flag_regex, mode, allow_team_switch, min_team_size, max_team_size,
       keep_scoreboard_frozen_after_end, created_at, updated_at
FROM competition
WHERE id = 1;

-- name: GetCompetitionForUpdate :one
SELECT id, name, start_time, end_time, freeze_time, is_paused, paused_at, is_public,
       flag_regex, mode, allow_team_switch, min_team_size, max_team_size,
       keep_scoreboard_frozen_after_end, created_at, updated_at
FROM competition
WHERE id = 1
FOR UPDATE;

-- name: UpdateCompetition :exec
UPDATE competition SET
    name = $1,
    start_time = $2,
    end_time = $3,
    freeze_time = $4,
    is_paused = $5,
    paused_at = $6,
    is_public = $7,
    flag_regex = $8,
    mode = $9,
    allow_team_switch = $10,
    min_team_size = $11,
    max_team_size = $12,
    keep_scoreboard_frozen_after_end = $13,
    updated_at = $14
WHERE id = 1;
