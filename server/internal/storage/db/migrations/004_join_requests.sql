CREATE TABLE IF NOT EXISTS join_requests (
    conv_id    VARCHAR(64) NOT NULL REFERENCES conversations(conv_id) ON DELETE CASCADE,
    user_id    VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status     SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (conv_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_join_requests_conv_status ON join_requests(conv_id, status);
