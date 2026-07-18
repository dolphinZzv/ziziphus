-- 032_add_user_conv_limit: Add conv_limit to users, update max_members default to 100
ALTER TABLE users ADD COLUMN IF NOT EXISTS conv_limit INT NOT NULL DEFAULT 100;
ALTER TABLE conversations ALTER COLUMN max_members SET DEFAULT 100;
