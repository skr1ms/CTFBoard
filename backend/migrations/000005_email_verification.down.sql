DROP INDEX idx_verification_user_type ON verification_tokens;
DROP INDEX idx_verification_token ON verification_tokens;
DROP TABLE verification_tokens;
ALTER TABLE users DROP COLUMN verified_at, DROP COLUMN is_verified;
