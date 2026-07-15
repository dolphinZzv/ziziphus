-- 025_drop_webhook_token: Remove token and require_audit columns
DROP INDEX IF EXISTS idx_conv_webhooks_token;
ALTER TABLE conv_webhooks DROP CONSTRAINT IF EXISTS conv_webhooks_token_key;
ALTER TABLE conv_webhooks DROP COLUMN IF EXISTS token;
ALTER TABLE conv_webhooks DROP COLUMN IF EXISTS require_audit;
