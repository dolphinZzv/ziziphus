-- conv_webhooks: webhook 配置表
CREATE TABLE IF NOT EXISTS conv_webhooks (
    id              BIGSERIAL PRIMARY KEY,
    conv_id         VARCHAR(64) NOT NULL REFERENCES conversations(conv_id) ON DELETE CASCADE,
    name            VARCHAR(128) NOT NULL,
    api_key_plain   VARCHAR(128) NOT NULL DEFAULT '',
    api_key_hash    VARCHAR(128) NOT NULL DEFAULT '',
    callback_url    VARCHAR(512) NOT NULL DEFAULT '',
    headers         JSONB DEFAULT '[]'::jsonb,
    cidr_whitelist  JSONB DEFAULT '[]'::jsonb,
    created_by      VARCHAR(32) NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(conv_id, name)
);
CREATE INDEX IF NOT EXISTS idx_conv_webhooks_conv_id ON conv_webhooks(conv_id);
