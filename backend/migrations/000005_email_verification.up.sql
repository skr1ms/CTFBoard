ALTER TABLE users
ADD COLUMN is_verified TINYINT(1) DEFAULT 0,
ADD COLUMN verified_at TIMESTAMP NULL;
CREATE TABLE verification_tokens (
    id CHAR(36) PRIMARY KEY,
    user_id CHAR(36) NOT NULL,
    token VARCHAR(64) NOT NULL UNIQUE,
    type ENUM(
        'email_verification',
        'password_reset'
    ) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_verification_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX idx_verification_token ON verification_tokens (token);
CREATE INDEX idx_verification_user_type ON verification_tokens (user_id, type);
