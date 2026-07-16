package config

import "fmt"

type Config struct {
	Server       ServerConfig       `mapstructure:"server"`
	Postgres     PostgresConfig     `mapstructure:"postgres"`
	Redis        RedisConfig        `mapstructure:"redis"`
	JWT          JWTConfig          `mapstructure:"jwt"`
	Snowflake    SnowflakeConfig    `mapstructure:"snowflake"`
	RateLimit    RateLimitConfig    `mapstructure:"ratelimit"`
	Storage      StorageConfig      `mapstructure:"storage"`
	SMTP         SMTPConfig         `mapstructure:"smtp"`
	Announcement AnnouncementConfig `mapstructure:"announcement"`
	Log          LogConfig          `mapstructure:"log"`
}

type AnnouncementConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Title   string `mapstructure:"title"`
	Body    string `mapstructure:"body"`
	URL     string `mapstructure:"url"`
}

type SMTPConfig struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	From     string `mapstructure:"from"`
}

type StorageConfig struct {
	LocalPath string `mapstructure:"local_path"`
	BaseURL   string `mapstructure:"base_url"`
}

type ServerConfig struct {
	Port              int  `mapstructure:"port"`
	AllowRegistration *bool `mapstructure:"allow_registration"`
}

func (s ServerConfig) RegistrationAllowed() bool {
	if s.AllowRegistration == nil {
		return true
	}
	return *s.AllowRegistration
}

type PostgresConfig struct {
	DSN        string `mapstructure:"dsn"`
	MaxConns   int    `mapstructure:"max_conns"`
	Migrations string `mapstructure:"migrations"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type JWTConfig struct {
	Secret             string `mapstructure:"secret"`
	ExpireHours        int    `mapstructure:"expire_hours"`
	RefreshExpireHours int    `mapstructure:"refresh_expire_hours"`
}

type SnowflakeConfig struct {
	WorkerID  int64  `mapstructure:"worker_id"`
	StartTime string `mapstructure:"start_time"`
}

type LogConfig struct {
	Level      string `mapstructure:"level"`
	File       string `mapstructure:"file"`
	MaxSize    int    `mapstructure:"max_size"`
	MaxAge     int    `mapstructure:"max_age"`
	MaxBackups int    `mapstructure:"max_backups"`
	Compress   bool   `mapstructure:"compress"`
}

type RateLimitConfig struct {
	MsgPerSec    int `mapstructure:"msg_per_sec"`
	MaxBodyBytes int `mapstructure:"max_body_bytes"`
	BurstSize    int `mapstructure:"burst_size"`

	APIEnabled      *bool `mapstructure:"api_enabled"`
	LoginAttempts   int   `mapstructure:"login_attempts"`
	LoginWindowMin  int   `mapstructure:"login_window_min"`
	LoginLockoutMin int   `mapstructure:"login_lockout_min"`
	RegPerWindow    int   `mapstructure:"reg_per_window"`
	RegWindowHour   int   `mapstructure:"reg_window_hour"`
	GlobalRate      int   `mapstructure:"global_rate"`
	GlobalBurst     int   `mapstructure:"global_burst"`
}

func (r RateLimitConfig) HTTPRateLimitEnabled() bool {
	if r.APIEnabled == nil {
		return true
	}
	return *r.APIEnabled
}

// Validate checks production-critical configuration values.
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

// Load is a convenience wrapper for backward compatibility.
// Prefer NewManager for production use.
func Load(path string) (*Config, error) {
	m, err := NewManager(path)
	if err != nil {
		return nil, err
	}
	return m.Get(), nil
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
		cfg.JWT.ExpireHours = 1
	}
	if cfg.JWT.RefreshExpireHours == 0 {
		cfg.JWT.RefreshExpireHours = 168
	}
	if cfg.RateLimit.MsgPerSec == 0 {
		cfg.RateLimit.MsgPerSec = 30
	}
	if cfg.RateLimit.MaxBodyBytes == 0 {
		cfg.RateLimit.MaxBodyBytes = 10240
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
