CREATE TABLE IF NOT EXISTS users (
    id          VARCHAR(32) PRIMARY KEY,
    type        SMALLINT NOT NULL DEFAULT 0,
    name        VARCHAR(128) NOT NULL,
    avatar      VARCHAR(256) NOT NULL DEFAULT '',
    status      SMALLINT NOT NULL DEFAULT 0,
    password    VARCHAR(512) NOT NULL,
    ext_meta    JSONB DEFAULT '{}',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_users_type ON users(type);
CREATE INDEX IF NOT EXISTS idx_users_name ON users(name);

CREATE TABLE IF NOT EXISTS sessions (
    session_id  VARCHAR(32) PRIMARY KEY,
    user_id     VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device      SMALLINT NOT NULL DEFAULT 0,
    device_name VARCHAR(128) NOT NULL DEFAULT '',
    status      SMALLINT NOT NULL DEFAULT 0,
    login_at    TIMESTAMP NOT NULL DEFAULT NOW(),
    last_active TIMESTAMP NOT NULL DEFAULT NOW(),
    metadata    JSONB DEFAULT '{}',
    expired_at  TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_status ON sessions(status);
CREATE INDEX IF NOT EXISTS idx_sessions_expired_at ON sessions(expired_at);

CREATE TABLE IF NOT EXISTS conversations (
    conv_id     VARCHAR(64) PRIMARY KEY,
    type        SMALLINT NOT NULL DEFAULT 1,
    name        VARCHAR(256) NOT NULL DEFAULT '',
    owner_id    VARCHAR(32) NOT NULL DEFAULT '',
    avatar      VARCHAR(256) NOT NULL DEFAULT '',
    max_members INT NOT NULL DEFAULT 200,
    last_msg_id BIGINT NOT NULL DEFAULT 0,
    last_msg_at TIMESTAMP,
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_conversations_last_msg ON conversations(last_msg_at DESC);

CREATE TABLE IF NOT EXISTS conv_members (
    conv_id   VARCHAR(64) NOT NULL REFERENCES conversations(conv_id) ON DELETE CASCADE,
    user_id   VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role      SMALLINT NOT NULL DEFAULT 0,
    nickname  VARCHAR(128) NOT NULL DEFAULT '',
    mute      BOOLEAN NOT NULL DEFAULT FALSE,
    joined_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (conv_id, user_id)
);
CREATE INDEX IF NOT EXISTS idx_conv_members_user_id ON conv_members(user_id);

CREATE TABLE IF NOT EXISTS messages (
    msg_id            BIGINT PRIMARY KEY,
    conv_id           VARCHAR(64) NOT NULL REFERENCES conversations(conv_id),
    sender_id         VARCHAR(32) NOT NULL,
    sender_session_id VARCHAR(32) NOT NULL DEFAULT '',
    content_type      SMALLINT NOT NULL DEFAULT 0,
    body              TEXT NOT NULL DEFAULT '',
    mention           TEXT[] DEFAULT '{}',
    reply_to          BIGINT NOT NULL DEFAULT 0,
    timestamp         BIGINT NOT NULL,
    client_seq        BIGINT NOT NULL DEFAULT 0,
    conv_seq          BIGINT NOT NULL DEFAULT 0,
    status            SMALLINT NOT NULL DEFAULT 1,
    deleted           BOOLEAN NOT NULL DEFAULT FALSE
);
CREATE INDEX IF NOT EXISTS idx_messages_conv_ts ON messages(conv_id, msg_id DESC);
CREATE UNIQUE INDEX IF NOT EXISTS idx_messages_dedup ON messages(sender_id, sender_session_id, client_seq);
CREATE INDEX IF NOT EXISTS idx_messages_body_gin ON messages USING GIN(to_tsvector('simple', body));

CREATE TABLE IF NOT EXISTS msg_receipts (
    msg_id    BIGINT NOT NULL REFERENCES messages(msg_id),
    user_id   VARCHAR(32) NOT NULL,
    session_id VARCHAR(32) NOT NULL DEFAULT '',
    status    SMALLINT NOT NULL DEFAULT 1,
    timestamp BIGINT NOT NULL,
    PRIMARY KEY (msg_id, session_id)
);
CREATE INDEX IF NOT EXISTS idx_receipts_user ON msg_receipts(user_id, msg_id);

CREATE TABLE IF NOT EXISTS contacts (
    user_id    VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    contact_id VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nickname   VARCHAR(128) NOT NULL DEFAULT '',
    added_at   TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, contact_id)
);
