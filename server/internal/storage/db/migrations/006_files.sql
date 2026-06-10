CREATE TABLE IF NOT EXISTS files (
    file_id      VARCHAR(64) PRIMARY KEY,
    uploader_id  VARCHAR(32) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR(512) NOT NULL,
    size         BIGINT NOT NULL DEFAULT 0,
    content_type SMALLINT NOT NULL DEFAULT 0,
    width        INT NOT NULL DEFAULT 0,
    height       INT NOT NULL DEFAULT 0,
    path         VARCHAR(1024) NOT NULL,
    thumbnail_path VARCHAR(1024) NOT NULL DEFAULT '',
    created_at   TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_files_uploader ON files(uploader_id);
CREATE INDEX IF NOT EXISTS idx_files_created ON files(created_at DESC);
