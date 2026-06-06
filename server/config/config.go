package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Postgres  PostgresConfig  `yaml:"postgres"`
	Redis     RedisConfig     `yaml:"redis"`
	JWT       JWTConfig       `yaml:"jwt"`
	Snowflake SnowflakeConfig `yaml:"snowflake"`
	RateLimit RateLimitConfig `yaml:"ratelimit"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

type PostgresConfig struct {
	DSN       string `yaml:"dsn"`
	MaxConns  int    `yaml:"max_conns"`
	Migrations string `yaml:"migrations"`
}

type RedisConfig struct {
	Addr     string `yaml:"addr"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWTConfig struct {
	Secret        string `yaml:"secret"`
	ExpireHours   int    `yaml:"expire_hours"`
	PrivateKeyPath string `yaml:"private_key_path"`
}

type SnowflakeConfig struct {
	WorkerID    int64  `yaml:"worker_id"`
	StartTime   string `yaml:"start_time"`
}

type RateLimitConfig struct {
	MsgPerSec    int   `yaml:"msg_per_sec"`
	MaxBodyBytes int   `yaml:"max_body_bytes"`
	BurstSize    int   `yaml:"burst_size"`
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
	return &cfg, nil
}

func setDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Postgres.MaxConns == 0 {
		cfg.Postgres.MaxConns = 20
	}
	if cfg.Postgres.Migrations == "" {
		cfg.Postgres.Migrations = "internal/storage/db/migrations"
	}
	if cfg.JWT.ExpireHours == 0 {
		cfg.JWT.ExpireHours = 168 // 7 days
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
}
