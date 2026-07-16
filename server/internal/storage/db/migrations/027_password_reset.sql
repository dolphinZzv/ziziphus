-- Password reset codes

CREATE TABLE IF NOT EXISTS password_reset (
    user_id    VARCHAR(32) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    code       VARCHAR(8) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
