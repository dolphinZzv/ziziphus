-- 030_add_file_visibility: Add visibility column to files table.
-- "public" files are served without auth; "private" files require auth + membership.
ALTER TABLE files ADD COLUMN IF NOT EXISTS visibility VARCHAR(16) NOT NULL DEFAULT 'public';
CREATE INDEX IF NOT EXISTS idx_files_visibility ON files(visibility);
