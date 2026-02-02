-- name: GetCompetition :one
SELECT id, name, start_time, end_time, freeze_time, is_paused, is_public,
       flag_regex, mode, allow_team_switch, min_team_size, max_team_size, created_at, updated_at
FROM competition
WHERE id = 1;

-- name: UpdateCompetition :exec
UPDATE competition SET
    name = $1,
    start_time = $2,
    end_time = $3,
    freeze_time = $4,
    is_paused = $5,
    is_public = $6,
    flag_regex = $7,
    mode = $8,
    allow_team_switch = $9,
    min_team_size = $10,
    max_team_size = $11,
    updated_at = $12
WHERE id = 1;
