CREATE TABLE IF NOT EXISTS email_verify (
    user_id      VARCHAR(32) PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    pending_email VARCHAR(256) NOT NULL,
    code         VARCHAR(6) NOT NULL,
    expires_at   TIMESTAMP NOT NULL
);
