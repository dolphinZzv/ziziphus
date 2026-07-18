package db

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/jackc/pgx/v5/pgxpool"
	"ziziphus/pkg/logger"
)

// Bootstrap migration — creates the tracking table. Always executed first.
const bootstrapMigration = `CREATE TABLE IF NOT EXISTS schema_migrations (
    filename    VARCHAR(255) PRIMARY KEY,
    applied_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    checksum    VARCHAR(64) NOT NULL DEFAULT ''
);`

func NewPgPool(ctx context.Context, dsn string, maxConns int) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse dsn: %w", err)
	}
	cfg.MaxConns = int32(maxConns)
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	return pool, nil
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	// 1. Read migration files first (fail fast on bad dir)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	// 2. Bootstrap: ensure schema_migrations table exists
	_, err = pool.Exec(ctx, bootstrapMigration)
	if err != nil {
		return fmt.Errorf("bootstrap schema_migrations: %w", err)
	}

	// 3. Read applied migration set
	applied := make(map[string]bool)
	rows, err := pool.Query(ctx, `SELECT filename FROM schema_migrations ORDER BY filename`)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err == nil {
				applied[name] = true
			}
		}
	}

	// 4. Migration table was just created and DB already has data →
	//    mark all existing migrations as applied to avoid re-running
	//    non-idempotent migrations on upgrade from the old runner.
	if len(applied) == 0 {
		// Only query if the table was likely just created
		var tableExists bool
		_ = pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'conversations')`,
		).Scan(&tableExists)
		if tableExists {
			logger.Info("migration table empty but DB has data — marking all migrations as applied")
			for _, entry := range entries {
				if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
					continue
				}
				if entry.Name() == "000_schema_migrations.sql" {
					continue
				}
				applied[entry.Name()] = true
				_, _ = pool.Exec(ctx,
					`INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
					entry.Name(), "migrated")
			}
			logger.Info("migration tracking initialized", "count", len(applied))
		}
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".sql" {
			continue
		}
		name := entry.Name()

		// Skip 000 bootstrap — already done above
		if name == "000_schema_migrations.sql" {
			continue
		}

		// Skip already-applied migrations
		if applied[name] {
			logger.Debug("migration already applied, skipping", "name", name)
			continue
		}

		// 4. Read and apply
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", name, err)
		}

		if _, err := pool.Exec(ctx, string(data)); err != nil {
			return fmt.Errorf("run migration %s: %w", name, err)
		}

		// 5. Record
		checksum := sha256.Sum256(data)
		cs := hex.EncodeToString(checksum[:])
		if _, err := pool.Exec(ctx,
			`INSERT INTO schema_migrations (filename, checksum) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			name, cs); err != nil {
			logger.Error("record migration failed", "name", name, "error", err)
		}

		logger.Info("migration applied", "name", name)
	}

	return nil
}
