-- 025_drop_webhook_token: Remove NOT NULL from token after dropping token usage
ALTER TABLE conv_webhooks ALTER COLUMN token DROP NOT NULL;
ALTER TABLE conv_webhooks ALTER COLUMN token SET DEFAULT '';
