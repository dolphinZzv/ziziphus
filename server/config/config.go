package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server       ServerConfig       `yaml:"server"`
	Postgres     PostgresConfig     `yaml:"postgres"`
	Redis        RedisConfig        `yaml:"redis"`
	JWT          JWTConfig          `yaml:"jwt"`
	Snowflake    SnowflakeConfig    `yaml:"snowflake"`
	RateLimit    RateLimitConfig    `yaml:"ratelimit"`
	Storage      StorageConfig      `yaml:"storage"`
	SMTP         SMTPConfig         `yaml:"smtp"`
	Announcement AnnouncementConfig `yaml:"announcement"`
	Log          LogConfig          `yaml:"log"`
}

type AnnouncementConfig struct {
	Enabled bool   `yaml:"enabled"`
	Title   string `yaml:"title"`
	Body    string `yaml:"body"`
	URL     string `yaml:"url"`
}

type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	From     string `yaml:"from"`
}

type StorageConfig struct {
	LocalPath string `yaml:"local_path"`
	BaseURL   string `yaml:"base_url"`
}

type ServerConfig struct {
	Port              int   `yaml:"port"`
	AllowRegistration *bool `yaml:"allow_registration"`
}

func (s ServerConfig) RegistrationAllowed() bool {
	if s.AllowRegistration == nil {
		return true // default: allow
	}
	return *s.AllowRegistration
}

type PostgresConfig struct {
	DSN        string `yaml:"dsn"`
	MaxConns   int    `yaml:"max_conns"`
	Migrations string `yaml:"migrations"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWTConfig struct {
	Secret             string `yaml:"secret"`
	ExpireHours        int    `yaml:"expire_hours"`
	RefreshExpireHours int    `yaml:"refresh_expire_hours"`
}

type SnowflakeConfig struct {
	WorkerID  int64  `yaml:"worker_id"`
	StartTime string `yaml:"start_time"`
}

// LogConfig controls logger behaviour.
type LogConfig struct {
	Level      string `yaml:"level"`       // debug, info, warn, error (default: info)
	File       string `yaml:"file"`        // log file path (empty = stdout only)
	MaxSize    int    `yaml:"max_size"`    // megabytes before rotation (default: 100)
	MaxAge     int    `yaml:"max_age"`     // days to retain old logs (default: 7)
	MaxBackups int    `yaml:"max_backups"` // number of old log files to retain (default: 10)
	Compress   bool   `yaml:"compress"`    // compress rotated files (default: true)
}

type RateLimitConfig struct {
	MsgPerSec    int `yaml:"msg_per_sec"`
	MaxBodyBytes int `yaml:"max_body_bytes"`
	BurstSize    int `yaml:"burst_size"`

	// HTTP API-level rate limiters (login, register, global per-IP DDoS).
	// Set api_enabled: false to disable all during E2E tests.
	APIEnabled      *bool `yaml:"api_enabled"`       // nil defaults to true
	LoginAttempts   int   `yaml:"login_attempts"`    // max failed login attempts per window
	LoginWindowMin  int   `yaml:"login_window_min"`  // window in minutes
	LoginLockoutMin int   `yaml:"login_lockout_min"` // lockout in minutes
	RegPerWindow    int   `yaml:"reg_per_window"`    // max registrations per window
	RegWindowHour   int   `yaml:"reg_window_hour"`   // window in hours
	GlobalRate      int   `yaml:"global_rate"`       // req/s per IP
	GlobalBurst     int   `yaml:"global_burst"`      // burst size
}

// HTTPRateLimitEnabled returns whether HTTP-level rate limiters are enabled.
// Defaults to true when the config key is absent (nil pointer).
func (r RateLimitConfig) HTTPRateLimitEnabled() bool {
	if r.APIEnabled == nil {
		return true
	}
	return *r.APIEnabled
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	setDefaults(&cfg)

	// Environment variable overrides (12-factor)
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		cfg.JWT.Secret = secret
	}

	return &cfg, nil
}

// Validate checks production-critical configuration values.
// Returns an error describing the first problem found.
func (c *Config) Validate() error {
	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long (got %d). "+
			"Generate one with: openssl rand -base64 32, then set JWT_SECRET env var or update config.yaml",
			len(c.JWT.Secret))
	}
	if c.SMTP.Host != "" && c.SMTP.User == "" {
		return fmt.Errorf("SMTP host set but user is empty")
	}
	return nil
}

func setDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Storage.LocalPath == "" {
		cfg.Storage.LocalPath = "data/files"
	}
	if cfg.Storage.BaseURL == "" {
		cfg.Storage.BaseURL = "/files"
	}
	if cfg.Postgres.MaxConns == 0 {
		cfg.Postgres.MaxConns = 20
	}
	if cfg.Postgres.Migrations == "" {
		cfg.Postgres.Migrations = "internal/storage/db/migrations"
	}
	if cfg.JWT.ExpireHours == 0 {
		cfg.JWT.ExpireHours = 1 // 1 hour (access token)
	}
	if cfg.JWT.RefreshExpireHours == 0 {
		cfg.JWT.RefreshExpireHours = 168 // 7 days (refresh token)
	}
	if cfg.RateLimit.MsgPerSec == 0 {
		cfg.RateLimit.MsgPerSec = 30
	}
	if cfg.RateLimit.MaxBodyBytes == 0 {
		cfg.RateLimit.MaxBodyBytes = 10240 // 10KB
	}
	if cfg.RateLimit.BurstSize == 0 {
		cfg.RateLimit.BurstSize = 50
	}
	if cfg.RateLimit.LoginAttempts == 0 {
		cfg.RateLimit.LoginAttempts = 10
	}
	if cfg.RateLimit.LoginWindowMin == 0 {
		cfg.RateLimit.LoginWindowMin = 1
	}
	if cfg.RateLimit.LoginLockoutMin == 0 {
		cfg.RateLimit.LoginLockoutMin = 5
	}
	if cfg.RateLimit.RegPerWindow == 0 {
		cfg.RateLimit.RegPerWindow = 5
	}
	if cfg.RateLimit.RegWindowHour == 0 {
		cfg.RateLimit.RegWindowHour = 1
	}
	if cfg.RateLimit.GlobalRate == 0 {
		cfg.RateLimit.GlobalRate = 100
	}
	if cfg.RateLimit.GlobalBurst == 0 {
		cfg.RateLimit.GlobalBurst = 200
	}
	if cfg.SMTP.Port == "" {
		cfg.SMTP.Port = "587"
	}
	if cfg.SMTP.From == "" {
		cfg.SMTP.From = cfg.SMTP.User
	}
	if cfg.Log.Level == "" {
		cfg.Log.Level = "info"
	}
	if cfg.Log.MaxSize == 0 {
		cfg.Log.MaxSize = 100
	}
	if cfg.Log.MaxAge == 0 {
		cfg.Log.MaxAge = 7
	}
	if cfg.Log.MaxBackups == 0 {
		cfg.Log.MaxBackups = 10
	}
}
