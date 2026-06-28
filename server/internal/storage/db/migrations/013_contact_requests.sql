-- 013_contact_requests: friend request table for bidirectional contact approval flow
CREATE TABLE IF NOT EXISTS contact_requests (
    id            BIGSERIAL PRIMARY KEY,
    from_user_id  VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_user_id    VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    form_msg_id   BIGINT NOT NULL DEFAULT 0,
    status        SMALLINT NOT NULL DEFAULT 0,  -- 0=pending, 1=approved, 2=rejected
    message       TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(from_user_id, to_user_id)
);

CREATE INDEX IF NOT EXISTS idx_contact_requests_to_user ON contact_requests(to_user_id, status);
CREATE INDEX IF NOT EXISTS idx_contact_requests_from_user ON contact_requests(from_user_id);
