-- 024_fs_folders: Replace DB folders with filesystem folders
DROP TABLE IF EXISTS file_folders;
ALTER TABLE files DROP COLUMN IF EXISTS folder_id;
ALTER TABLE files ADD COLUMN IF NOT EXISTS folder_path VARCHAR(512) NOT NULL DEFAULT '';
