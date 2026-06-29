-- conv_webhooks: webhook 配置表
CREATE TABLE IF NOT EXISTS conv_webhooks (
    id              BIGSERIAL PRIMARY KEY,
    conv_id         VARCHAR(64) NOT NULL REFERENCES conversations(conv_id) ON DELETE CASCADE,
    name            VARCHAR(128) NOT NULL,
    token           VARCHAR(64) NOT NULL UNIQUE,
    api_key_hash    VARCHAR(128) NOT NULL DEFAULT '',
    callback_url    VARCHAR(512) NOT NULL DEFAULT '',
    headers         JSONB DEFAULT '[]'::jsonb,
    cidr_whitelist  JSONB DEFAULT '[]'::jsonb,
    require_audit   BOOLEAN NOT NULL DEFAULT false,
    created_by      VARCHAR(32) NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(conv_id, name)
);
CREATE INDEX IF NOT EXISTS idx_conv_webhooks_conv_id ON conv_webhooks(conv_id);
CREATE INDEX IF NOT EXISTS idx_conv_webhooks_token ON conv_webhooks(token);

-- webhook_audit_logs: 审计日志
CREATE TABLE IF NOT EXISTS webhook_audit_logs (
    id          BIGSERIAL PRIMARY KEY,
    webhook_id  BIGINT NOT NULL REFERENCES conv_webhooks(id) ON DELETE CASCADE,
    conv_id     VARCHAR(64) NOT NULL,
    msg_id      BIGINT NOT NULL,
    action      VARCHAR(16) NOT NULL,
    actor_id    VARCHAR(32) NOT NULL DEFAULT '',
    reason      VARCHAR(256) DEFAULT '',
    caller_ip   VARCHAR(45) DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_webhook_audit_logs_conv ON webhook_audit_logs(conv_id);
CREATE INDEX IF NOT EXISTS idx_webhook_audit_logs_wh ON webhook_audit_logs(webhook_id);

-- webhook_messages: webhook 发出的消息关联
CREATE TABLE IF NOT EXISTS webhook_messages (
    msg_id      BIGINT PRIMARY KEY REFERENCES messages(msg_id) ON DELETE CASCADE,
    webhook_id  BIGINT NOT NULL REFERENCES conv_webhooks(id) ON DELETE CASCADE,
    conv_id     VARCHAR(64) NOT NULL,
    audit_status VARCHAR(16) NOT NULL DEFAULT '',
    source_ip   VARCHAR(45) NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_webhook_messages_audit ON webhook_messages(audit_status, conv_id);
CREATE INDEX IF NOT EXISTS idx_webhook_messages_wh ON webhook_messages(webhook_id);
CREATE INDEX IF NOT EXISTS idx_webhook_messages_conv ON webhook_messages(conv_id);
