-- 033_oauth: Add GitHub/Google OAuth ID columns
--
-- Up
ALTER TABLE users ADD COLUMN IF NOT EXISTS github_id VARCHAR(128) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_id VARCHAR(128) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_users_github_id ON users(github_id);
CREATE INDEX IF NOT EXISTS idx_users_google_id ON users(google_id);

-- Down
-- ALTER TABLE users DROP COLUMN IF EXISTS github_id;
-- ALTER TABLE users DROP COLUMN IF EXISTS google_id;
