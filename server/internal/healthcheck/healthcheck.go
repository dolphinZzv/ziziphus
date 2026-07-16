package healthcheck

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Check verifies PostgreSQL and Redis are reachable, logs each result,
// and returns an error if either dependency is unavailable.
func Check(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client) error {
	timeout := 5 * time.Second

	// PostgreSQL
	slog.Info("checking PostgreSQL...")
	pgCtx, pgCancel := context.WithTimeout(ctx, timeout)
	defer pgCancel()
	if err := pool.Ping(pgCtx); err != nil {
		slog.Error("PostgreSQL check failed", "error", err)
		return fmt.Errorf("postgresql: %w", err)
	}
	slog.Info("PostgreSQL ok")

	// Redis
	slog.Info("checking Redis...")
	redisCtx, redisCancel := context.WithTimeout(ctx, timeout)
	defer redisCancel()
	if err := rdb.Ping(redisCtx).Err(); err != nil {
		slog.Error("Redis check failed", "error", err)
		return fmt.Errorf("redis: %w", err)
	}
	slog.Info("Redis ok")

	return nil
}
