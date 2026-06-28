-- 017_conv_files: Add conv_id to files table for per-conversation file directories
ALTER TABLE files ADD COLUMN IF NOT EXISTS conv_id VARCHAR(64) NOT NULL DEFAULT '';
CREATE INDEX IF NOT EXISTS idx_files_conv_id ON files(conv_id);
