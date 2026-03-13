-- +goose Up
INSERT INTO configs (key, value, value_type, description, category) VALUES
('ctf_name', 'AstroCTFb', 'string', 'CTF competition name', 'general'),
('ctf_description', '', 'string', 'CTF competition description (Markdown)', 'general'),
('ctf_logo', '', 'string', 'CTF logo URL', 'general'),
('user_mode', 'teams', 'string', 'Participation mode: teams or users', 'general'),
('theme_color_primary', '#6366f1', 'string', 'Primary theme color (hex)', 'theme'),
('theme_color_secondary', '#4f46e5', 'string', 'Secondary theme color (hex)', 'theme'),
('theme_header_html', '', 'string', 'Custom HTML for <head>', 'theme'),
('theme_footer_html', '', 'string', 'Custom HTML for footer', 'theme'),
('theme_dark_mode', 'true', 'bool', 'Enable dark theme by default', 'theme'),
('challenge_visibility', 'private', 'string', 'Challenge visibility: public, private, admins', 'visibility'),
('score_visibility', 'public', 'string', 'Scoreboard visibility: public, hidden, admins', 'visibility'),
('registration_visibility', 'public', 'string', 'Registration visibility: public, private', 'visibility'),
('view_after_ctf', 'true', 'bool', 'Show challenges after CTF ends', 'visibility'),
('scoring_type', 'dynamic', 'string', 'Scoring type: static or dynamic', 'scoring'),
('decay_function', 'linear', 'string', 'Decay function: linear or logarithmic', 'scoring'),
('first_blood_bonus', '0', 'int', 'Bonus points for first blood', 'scoring'),
('mail_verification_subject', 'Verify your email — {ctf_name}', 'string', 'Verification email subject (supports {ctf_name})', 'email'),
('mail_verification_body', 'Follow the link to verify: {url}', 'string', 'Verification email body (supports {url}, {ctf_name})', 'email'),
('mail_reset_subject', 'Password reset — {ctf_name}', 'string', 'Password reset email subject', 'email'),
('mail_reset_body', 'Follow the link to reset password: {url}', 'string', 'Password reset email body', 'email'),
('social_github', '', 'string', 'CTF GitHub account URL', 'social'),
('social_discord', '', 'string', 'Discord server URL', 'social'),
('social_twitter', '', 'string', 'Twitter/X account URL', 'social'),
('social_website', '', 'string', 'Organizers website URL', 'social'),
('tos_url', '', 'string', 'Terms of service URL (external link)', 'legal'),
('tos_text', '', 'string', 'Terms of service text (Markdown)', 'legal'),
('privacy_url', '', 'string', 'Privacy policy URL', 'legal'),
('privacy_text', '', 'string', 'Privacy policy text (Markdown)', 'legal'),
('html_sanitization', 'true', 'bool', 'Sanitize HTML in challenge descriptions', 'advanced'),
('csv_delimiter', ',', 'string', 'CSV export delimiter', 'advanced'),
('max_content_length', '10485760', 'int', 'Max content size in bytes (10MB)', 'advanced'),
('registration_code', '', 'string', 'Registration access code (empty = disabled)', 'advanced'),
('domain_whitelist', '', 'string', 'Allowed email domains (comma-separated, empty = all)', 'advanced'),
('password_min_length', '8', 'int', 'Minimum password length', 'advanced')
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DELETE FROM configs WHERE key IN (
'ctf_name','ctf_description','ctf_logo','user_mode',
'theme_color_primary','theme_color_secondary','theme_header_html','theme_footer_html','theme_dark_mode',
'challenge_visibility','score_visibility','registration_visibility','view_after_ctf',
'scoring_type','decay_function','first_blood_bonus',
'mail_verification_subject','mail_verification_body','mail_reset_subject','mail_reset_body',
'social_github','social_discord','social_twitter','social_website',
'tos_url','tos_text','privacy_url','privacy_text',
'html_sanitization','csv_delimiter','max_content_length','registration_code','domain_whitelist','password_min_length'
);
