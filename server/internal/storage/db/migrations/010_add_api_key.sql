ALTER TABLE users ADD COLUMN IF NOT EXISTS api_key TEXT NOT NULL DEFAULT '';
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_api_key ON users(api_key) WHERE api_key != '';
