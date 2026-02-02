-- name: GetAppSettings :one
SELECT id, app_name, verify_emails, frontend_url, cors_origins,
       resend_enabled, resend_from_email, resend_from_name,
       verify_ttl_hours, reset_ttl_hours,
       submit_limit_per_user, submit_limit_duration_min,
       scoreboard_visible, registration_open, updated_at
FROM app_settings
WHERE id = 1;

-- name: UpdateAppSettings :exec
UPDATE app_settings SET
    app_name = $1,
    verify_emails = $2,
    frontend_url = $3,
    cors_origins = $4,
    resend_enabled = $5,
    resend_from_email = $6,
    resend_from_name = $7,
    verify_ttl_hours = $8,
    reset_ttl_hours = $9,
    submit_limit_per_user = $10,
    submit_limit_duration_min = $11,
    scoreboard_visible = $12,
    registration_open = $13,
    updated_at = $14
WHERE id = 1;
