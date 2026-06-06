-- name: GetAppSettings :one
SELECT id, app_name, verify_emails, frontend_url, cors_origins,
       resend_enabled, resend_from_email, resend_from_name,
       verify_ttl_hours, reset_ttl_hours,
       submit_limit_per_user, submit_limit_duration_min,
       scoreboard_visible, registration_open,
       default_per_page, max_per_page, csv_export_max_rows,
       rate_limit_login_per_minute, rate_limit_register_per_minute,
       rate_limit_forgot_password_per_minute, rate_limit_reset_password_per_minute,
       rate_limit_logout_per_minute, rate_limit_refresh_per_minute,
       rate_limit_scoreboard_per_minute, rate_limit_general_ip_per_minute,
       rate_limit_verify_email_per_minute, rate_limit_oauth_callback_per_minute,
       rate_limit_oauth_redirect_per_minute,
       rate_limit_comment_per_minute,
       max_teams, writeup_enabled, oauth_github_enabled, oauth_google_enabled, updated_at
FROM app_settings
WHERE id = 1;

-- name: GetAppSettingsForUpdate :one
SELECT id, app_name, verify_emails, frontend_url, cors_origins,
       resend_enabled, resend_from_email, resend_from_name,
       verify_ttl_hours, reset_ttl_hours,
       submit_limit_per_user, submit_limit_duration_min,
       scoreboard_visible, registration_open,
       default_per_page, max_per_page, csv_export_max_rows,
       rate_limit_login_per_minute, rate_limit_register_per_minute,
       rate_limit_forgot_password_per_minute, rate_limit_reset_password_per_minute,
       rate_limit_logout_per_minute, rate_limit_refresh_per_minute,
       rate_limit_scoreboard_per_minute, rate_limit_general_ip_per_minute,
       rate_limit_verify_email_per_minute, rate_limit_oauth_callback_per_minute,
       rate_limit_oauth_redirect_per_minute,
       rate_limit_comment_per_minute,
       max_teams, writeup_enabled, oauth_github_enabled, oauth_google_enabled, updated_at
FROM app_settings
WHERE id = 1
FOR UPDATE;

-- name: ReconcileAppSettingsStartupDefaults :exec
UPDATE app_settings
SET app_name = $1,
    resend_from_name = $2,
    resend_from_email = $3,
    resend_enabled = $4,
    oauth_github_enabled = $5,
    oauth_google_enabled = $6
WHERE id = 1
  AND app_name = $7;

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
    default_per_page = $14,
    max_per_page = $15,
    csv_export_max_rows = $16,
    rate_limit_login_per_minute = $17,
    rate_limit_register_per_minute = $18,
    rate_limit_forgot_password_per_minute = $19,
    rate_limit_reset_password_per_minute = $20,
    rate_limit_logout_per_minute = $21,
    rate_limit_refresh_per_minute = $22,
    rate_limit_scoreboard_per_minute = $23,
    rate_limit_general_ip_per_minute = $24,
    rate_limit_verify_email_per_minute = $25,
    rate_limit_oauth_callback_per_minute = $26,
    rate_limit_oauth_redirect_per_minute = $27,
    rate_limit_comment_per_minute = $28,
    max_teams = $29,
    writeup_enabled = $30,
    oauth_github_enabled = $31,
    oauth_google_enabled = $32,
    updated_at = $33
WHERE id = 1;

-- name: UpdateAppSettingsIfCurrent :one
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
    default_per_page = $14,
    max_per_page = $15,
    csv_export_max_rows = $16,
    rate_limit_login_per_minute = $17,
    rate_limit_register_per_minute = $18,
    rate_limit_forgot_password_per_minute = $19,
    rate_limit_reset_password_per_minute = $20,
    rate_limit_logout_per_minute = $21,
    rate_limit_refresh_per_minute = $22,
    rate_limit_scoreboard_per_minute = $23,
    rate_limit_general_ip_per_minute = $24,
    rate_limit_verify_email_per_minute = $25,
    rate_limit_oauth_callback_per_minute = $26,
    rate_limit_oauth_redirect_per_minute = $27,
    rate_limit_comment_per_minute = $28,
    max_teams = $29,
    writeup_enabled = $30,
    oauth_github_enabled = $31,
    oauth_google_enabled = $32,
    updated_at = $33
WHERE id = 1 AND updated_at = $34
RETURNING id;
