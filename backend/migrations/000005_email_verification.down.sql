DROP INDEX IF EXISTS idx_verification_user_type;
DROP INDEX IF EXISTS idx_verification_token;
DROP TABLE IF EXISTS verification_tokens;
ALTER TABLE users DROP COLUMN IF EXISTS verified_at, DROP COLUMN IF EXISTS is_verified;
