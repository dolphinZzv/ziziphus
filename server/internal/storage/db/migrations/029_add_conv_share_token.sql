-- Add share_token column for public group cards
ALTER TABLE conversations ADD COLUMN IF NOT EXISTS share_token VARCHAR(32) DEFAULT '';
-- Use a partial unique index so empty strings don't conflict
DROP INDEX IF EXISTS idx_conversations_share_token;
CREATE UNIQUE INDEX IF NOT EXISTS idx_conversations_share_token ON conversations(share_token) WHERE share_token != '';
