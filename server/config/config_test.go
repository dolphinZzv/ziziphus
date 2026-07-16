package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "valid.yaml")
	content := []byte(`
server:
  port: 9090
postgres:
  dsn: "postgres://localhost:5432/testdb"
  max_conns: 50
  migrations: "custom/path"
redis:
  addr: "redis:6379"
  password: "redis-secret"
  db: 2
jwt:
  secret: "test-secret"
  expire_hours: 72
  refresh_expire_hours: 168
snowflake:
  worker_id: 3
  start_time: "2025-01-01T00:00:00Z"
ratelimit:
  msg_per_sec: 100
  max_body_bytes: 20480
  burst_size: 200
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Server
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want 9090", cfg.Server.Port)
	}

	// Postgres
	if cfg.Postgres.DSN != "postgres://localhost:5432/testdb" {
		t.Errorf("Postgres.DSN = %q, want %q", cfg.Postgres.DSN, "postgres://localhost:5432/testdb")
	}
	if cfg.Postgres.MaxConns != 50 {
		t.Errorf("Postgres.MaxConns = %d, want 50", cfg.Postgres.MaxConns)
	}
	if cfg.Postgres.Migrations != "custom/path" {
		t.Errorf("Postgres.Migrations = %q, want %q", cfg.Postgres.Migrations, "custom/path")
	}

	// Redis
	if cfg.Redis.Addr != "redis:6379" {
		t.Errorf("Redis.Addr = %q, want %q", cfg.Redis.Addr, "redis:6379")
	}
	if cfg.Redis.Password != "redis-secret" {
		t.Errorf("Redis.Password = %q, want %q", cfg.Redis.Password, "redis-secret")
	}
	if cfg.Redis.DB != 2 {
		t.Errorf("Redis.DB = %d, want 2", cfg.Redis.DB)
	}

	// JWT
	if cfg.JWT.Secret != "test-secret" {
		t.Errorf("JWT.Secret = %q, want %q", cfg.JWT.Secret, "test-secret")
	}
	if cfg.JWT.ExpireHours != 72 {
		t.Errorf("JWT.ExpireHours = %d, want 72", cfg.JWT.ExpireHours)
	}
	if cfg.JWT.RefreshExpireHours != 168 {
		t.Errorf("JWT.RefreshExpireHours = %d, want 168", cfg.JWT.RefreshExpireHours)
	}

	// Snowflake
	if cfg.Snowflake.WorkerID != 3 {
		t.Errorf("Snowflake.WorkerID = %d, want 3", cfg.Snowflake.WorkerID)
	}
	if cfg.Snowflake.StartTime != "2025-01-01T00:00:00Z" {
		t.Errorf("Snowflake.StartTime = %q, want %q", cfg.Snowflake.StartTime, "2025-01-01T00:00:00Z")
	}

	// RateLimit
	if cfg.RateLimit.MsgPerSec != 100 {
		t.Errorf("RateLimit.MsgPerSec = %d, want 100", cfg.RateLimit.MsgPerSec)
	}
	if cfg.RateLimit.MaxBodyBytes != 20480 {
		t.Errorf("RateLimit.MaxBodyBytes = %d, want 20480", cfg.RateLimit.MaxBodyBytes)
	}
	if cfg.RateLimit.BurstSize != 200 {
		t.Errorf("RateLimit.BurstSize = %d, want 200", cfg.RateLimit.BurstSize)
	}
}

func TestLoad_NonExistentFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Load() expected error for non-existent file, got nil")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "invalid.yaml")
	content := []byte("server:\n  port: not-a-number\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() expected error for invalid YAML, got nil")
	}
}

func TestLoad_MinimalConfig_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minimal.yaml")
	// Write an empty YAML file — all fields will be zero-valued.
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load() returned nil config")
	}

	// Verify defaults are applied
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port default = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Postgres.MaxConns != 20 {
		t.Errorf("Postgres.MaxConns default = %d, want 20", cfg.Postgres.MaxConns)
	}
	if cfg.Postgres.Migrations != "internal/storage/db/migrations" {
		t.Errorf("Postgres.Migrations default = %q, want %q", cfg.Postgres.Migrations, "internal/storage/db/migrations")
	}
	if cfg.JWT.ExpireHours != 1 {
		t.Errorf("JWT.ExpireHours default = %d, want 1", cfg.JWT.ExpireHours)
	}
	if cfg.JWT.RefreshExpireHours != 168 {
		t.Errorf("JWT.RefreshExpireHours default = %d, want 168", cfg.JWT.RefreshExpireHours)
	}
	if cfg.RateLimit.MsgPerSec != 30 {
		t.Errorf("RateLimit.MsgPerSec default = %d, want 30", cfg.RateLimit.MsgPerSec)
	}
	if cfg.RateLimit.MaxBodyBytes != 10240 {
		t.Errorf("RateLimit.MaxBodyBytes default = %d, want 10240", cfg.RateLimit.MaxBodyBytes)
	}
	if cfg.RateLimit.BurstSize != 50 {
		t.Errorf("RateLimit.BurstSize default = %d, want 50", cfg.RateLimit.BurstSize)
	}
}

func TestServerConfig_RegistrationAllowed(t *testing.T) {
	t.Run("nil AllowRegistration returns true", func(t *testing.T) {
		s := ServerConfig{AllowRegistration: nil}
		if !s.RegistrationAllowed() {
			t.Error("RegistrationAllowed() = false, want true")
		}
	})

	t.Run("true AllowRegistration returns true", func(t *testing.T) {
		v := true
		s := ServerConfig{AllowRegistration: &v}
		if !s.RegistrationAllowed() {
			t.Error("RegistrationAllowed() = false, want true")
		}
	})

	t.Run("false AllowRegistration returns false", func(t *testing.T) {
		v := false
		s := ServerConfig{AllowRegistration: &v}
		if s.RegistrationAllowed() {
			t.Error("RegistrationAllowed() = true, want false")
		}
	})
}

func TestRateLimitConfig_HTTPRateLimitEnabled(t *testing.T) {
	t.Run("nil APIEnabled returns true", func(t *testing.T) {
		r := RateLimitConfig{APIEnabled: nil}
		if !r.HTTPRateLimitEnabled() {
			t.Error("HTTPRateLimitEnabled() = false, want true")
		}
	})

	t.Run("true APIEnabled returns true", func(t *testing.T) {
		v := true
		r := RateLimitConfig{APIEnabled: &v}
		if !r.HTTPRateLimitEnabled() {
			t.Error("HTTPRateLimitEnabled() = false, want true")
		}
	})

	t.Run("false APIEnabled returns false", func(t *testing.T) {
		v := false
		r := RateLimitConfig{APIEnabled: &v}
		if r.HTTPRateLimitEnabled() {
			t.Error("HTTPRateLimitEnabled() = true, want false")
		}
	})
}

func TestConfig_Validate(t *testing.T) {
	t.Run("short JWT secret returns error", func(t *testing.T) {
		c := &Config{JWT: JWTConfig{Secret: "short"}}
		err := c.Validate()
		if err == nil {
			t.Fatal("Validate() expected error for short JWT secret, got nil")
		}
	})

	t.Run("empty SMTP host and user is valid", func(t *testing.T) {
		c := &Config{JWT: JWTConfig{Secret: "this-is-a-long-enough-secret-key-32+"}}
		err := c.Validate()
		if err != nil {
			t.Fatalf("Validate() = %v, want nil", err)
		}
	})

	t.Run("SMTP host set but user empty returns error", func(t *testing.T) {
		c := &Config{
			JWT:  JWTConfig{Secret: "this-is-a-long-enough-secret-key-32+"},
			SMTP: SMTPConfig{Host: "smtp.example.com", User: ""},
		}
		err := c.Validate()
		if err == nil {
			t.Fatal("Validate() expected error for SMTP host without user, got nil")
		}
	})

	t.Run("SMTP host and user both set is valid", func(t *testing.T) {
		c := &Config{
			JWT:  JWTConfig{Secret: "this-is-a-long-enough-secret-key-32+"},
			SMTP: SMTPConfig{Host: "smtp.example.com", User: "user"},
		}
		err := c.Validate()
		if err != nil {
			t.Fatalf("Validate() = %v, want nil", err)
		}
	})
}

func TestSetDefaults(t *testing.T) {
	t.Run("non-zero values are not overwritten", func(t *testing.T) {
		cfg := &Config{
			Server: ServerConfig{Port: 3000},
			Postgres: PostgresConfig{
				MaxConns:   42,
				Migrations: "custom/migrations",
			},
			JWT: JWTConfig{ExpireHours: 24},
			RateLimit: RateLimitConfig{
				MsgPerSec:    99,
				MaxBodyBytes: 9999,
				BurstSize:    88,
			},
		}
		setDefaults(cfg)

		if cfg.Server.Port != 3000 {
			t.Errorf("Server.Port = %d, want 3000", cfg.Server.Port)
		}
		if cfg.Postgres.MaxConns != 42 {
			t.Errorf("Postgres.MaxConns = %d, want 42", cfg.Postgres.MaxConns)
		}
		if cfg.Postgres.Migrations != "custom/migrations" {
			t.Errorf("Postgres.Migrations = %q, want %q", cfg.Postgres.Migrations, "custom/migrations")
		}
		if cfg.JWT.ExpireHours != 24 {
			t.Errorf("JWT.ExpireHours = %d, want 24", cfg.JWT.ExpireHours)
		}
		if cfg.JWT.RefreshExpireHours != 168 {
			t.Errorf("JWT.RefreshExpireHours = %d, want 168", cfg.JWT.RefreshExpireHours)
		}
		if cfg.RateLimit.MsgPerSec != 99 {
			t.Errorf("RateLimit.MsgPerSec = %d, want 99", cfg.RateLimit.MsgPerSec)
		}
		if cfg.RateLimit.MaxBodyBytes != 9999 {
			t.Errorf("RateLimit.MaxBodyBytes = %d, want 9999", cfg.RateLimit.MaxBodyBytes)
		}
		if cfg.RateLimit.BurstSize != 88 {
			t.Errorf("RateLimit.BurstSize = %d, want 88", cfg.RateLimit.BurstSize)
		}
	})

	t.Run("defaults for all default values", func(t *testing.T) {
		cfg := &Config{}
		setDefaults(cfg)

		if cfg.Server.Port != 8080 {
			t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
		}
		if cfg.Postgres.MaxConns != 20 {
			t.Errorf("Postgres.MaxConns = %d, want 20", cfg.Postgres.MaxConns)
		}
		if cfg.Postgres.Migrations != "internal/storage/db/migrations" {
			t.Errorf("Postgres.Migrations = %q, want %q", cfg.Postgres.Migrations, "internal/storage/db/migrations")
		}
		if cfg.JWT.ExpireHours != 1 {
			t.Errorf("JWT.ExpireHours = %d, want 1", cfg.JWT.ExpireHours)
		}
		if cfg.JWT.RefreshExpireHours != 168 {
			t.Errorf("JWT.RefreshExpireHours = %d, want 168", cfg.JWT.RefreshExpireHours)
		}
		if cfg.RateLimit.MsgPerSec != 30 {
			t.Errorf("RateLimit.MsgPerSec = %d, want 30", cfg.RateLimit.MsgPerSec)
		}
		if cfg.RateLimit.MaxBodyBytes != 10240 {
			t.Errorf("RateLimit.MaxBodyBytes = %d, want 10240", cfg.RateLimit.MaxBodyBytes)
		}
		if cfg.RateLimit.BurstSize != 50 {
			t.Errorf("RateLimit.BurstSize = %d, want 50", cfg.RateLimit.BurstSize)
		}
	})
}
