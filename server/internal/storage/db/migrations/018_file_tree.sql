-- 018_file_tree: Add folder support to file system
CREATE TABLE IF NOT EXISTS file_folders (
    folder_id   BIGSERIAL PRIMARY KEY,
    conv_id     VARCHAR(64) NOT NULL DEFAULT '',
    name        VARCHAR(256) NOT NULL DEFAULT '',
    parent_id   BIGINT NOT NULL DEFAULT 0,
    created_by  VARCHAR(32) NOT NULL DEFAULT '',
    created_at  TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_file_folders_conv ON file_folders(conv_id);
CREATE INDEX IF NOT EXISTS idx_file_folders_parent ON file_folders(parent_id);

ALTER TABLE files ADD COLUMN IF NOT EXISTS folder_id BIGINT NOT NULL DEFAULT 0;
CREATE INDEX IF NOT EXISTS idx_files_folder ON files(folder_id);
