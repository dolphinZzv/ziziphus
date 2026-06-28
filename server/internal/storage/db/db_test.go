package db

import (
	"context"
	"testing"
)

func TestNewPgPool_InvalidDSN(t *testing.T) {
	_, err := NewPgPool(context.Background(), "invalid-dsn", 10)
	if err == nil {
		t.Fatal("expected error for invalid DSN, got nil")
	}
}

func TestRunMigrations_InvalidDir(t *testing.T) {
	err := RunMigrations(context.Background(), nil, "/nonexistent/migrations")
	if err == nil {
		t.Fatal("expected error for invalid directory, got nil")
	}
}
