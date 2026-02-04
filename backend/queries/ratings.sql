-- name: CreateCTFEvent :one
INSERT INTO ctf_events (id, name, start_time, end_time, weight, created_at)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id, name, start_time, end_time, weight, created_at;

-- name: GetCTFEventByID :one
SELECT id, name, start_time, end_time, weight, created_at
FROM ctf_events WHERE id = $1;

-- name: GetAllCTFEvents :many
SELECT id, name, start_time, end_time, weight, created_at
FROM ctf_events ORDER BY start_time DESC;

-- name: CreateTeamRating :exec
INSERT INTO team_ratings (id, team_id, ctf_event_id, rank, score, rating_points, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetTeamRatingsByTeamID :many
SELECT id, team_id, ctf_event_id, rank, score, rating_points, created_at
FROM team_ratings WHERE team_id = $1 ORDER BY created_at DESC;

-- name: UpsertGlobalRating :exec
INSERT INTO global_ratings (team_id, total_points, events_count, best_rank, last_updated)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (team_id) DO UPDATE SET
    total_points = EXCLUDED.total_points,
    events_count = EXCLUDED.events_count,
    best_rank = EXCLUDED.best_rank,
    last_updated = EXCLUDED.last_updated;

-- name: GetGlobalRatings :many
SELECT gr.team_id, t.name AS team_name, gr.total_points, gr.events_count, gr.best_rank, gr.last_updated
FROM global_ratings gr
JOIN teams t ON t.id = gr.team_id
WHERE t.deleted_at IS NULL AND t.is_banned = FALSE AND t.is_hidden = FALSE
ORDER BY gr.total_points DESC, gr.last_updated DESC
LIMIT $1 OFFSET $2;

-- name: CountGlobalRatings :one
SELECT COUNT(*) FROM global_ratings gr
JOIN teams t ON t.id = gr.team_id
WHERE t.deleted_at IS NULL AND t.is_banned = FALSE AND t.is_hidden = FALSE;

-- name: GetGlobalRatingByTeamID :one
SELECT gr.team_id, t.name AS team_name, gr.total_points, gr.events_count, gr.best_rank, gr.last_updated
FROM global_ratings gr
JOIN teams t ON t.id = gr.team_id
WHERE gr.team_id = $1;
