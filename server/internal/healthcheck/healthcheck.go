package healthcheck

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/jackc/pgx/v5/pgxpool"
	"ziziphus/pkg/logger"
)

// Check verifies PostgreSQL and Redis are reachable, logs each result,
// and returns an error if either dependency is unavailable.
func Check(ctx context.Context, pool *pgxpool.Pool, rdb *redis.Client) error {
	timeout := 5 * time.Second

	// PostgreSQL
	logger.Info("checking PostgreSQL...")
	pgCtx, pgCancel := context.WithTimeout(ctx, timeout)
	defer pgCancel()
	if err := pool.Ping(pgCtx); err != nil {
		logger.Error("PostgreSQL check failed", "error", err)
		return fmt.Errorf("postgresql: %w", err)
	}
	logger.Info("PostgreSQL ok")

	// Redis
	logger.Info("checking Redis...")
	redisCtx, redisCancel := context.WithTimeout(ctx, timeout)
	defer redisCancel()
	if err := rdb.Ping(redisCtx).Err(); err != nil {
		logger.Error("Redis check failed", "error", err)
		return fmt.Errorf("redis: %w", err)
	}
	logger.Info("Redis ok")

	return nil
}
