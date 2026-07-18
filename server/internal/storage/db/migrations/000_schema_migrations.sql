-- schema_migrations tracks which migrations have been applied.
-- This table is created first by the migration runner.
CREATE TABLE IF NOT EXISTS schema_migrations (
    filename    VARCHAR(255) PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    checksum    VARCHAR(64) NOT NULL DEFAULT ''   -- SHA-256 of file content
);
