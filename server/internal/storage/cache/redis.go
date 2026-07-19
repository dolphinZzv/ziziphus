package cache

import (
	"context"

	"github.com/go-redis/redis/extra/redisotel/v8"
	"github.com/go-redis/redis/v8"
)

// NewRedisClient creates a Redis client with OTel tracing instrumentation.
func NewRedisClient(addr, password string, db int) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	// Instrument with OpenTelemetry tracing so every Redis command produces a
	// child span under the parent request's trace.
	client.AddHook(redisotel.NewTracingHook())

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}
	return client, nil
}
