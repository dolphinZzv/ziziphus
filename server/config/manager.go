package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/spf13/viper"
)

// Manager wraps viper for configuration management with hot-reload support.
type Manager struct {
	v        *viper.Viper
	mu       sync.RWMutex
	current  *Config
	onChange []func(*Config)
}

// NewManager loads configuration from the given YAML path via viper.
func NewManager(path string) (*Manager, error) {
	v := viper.New()
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	// Map env vars to config keys: JWT_SECRET → jwt.secret
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	m := &Manager{v: v}
	if err := m.reload(); err != nil {
		return nil, err
	}
	return m, nil
}

// Get returns the current config snapshot (thread-safe).
func (m *Manager) Get() *Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

// OnChange registers a callback fired after each successful reload.
func (m *Manager) OnChange(fn func(*Config)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.onChange = append(m.onChange, fn)
}

// Reload re-reads the config file and triggers change callbacks.
func (m *Manager) Reload() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if err := m.v.ReadInConfig(); err != nil {
		return fmt.Errorf("re-read config: %w", err)
	}
	return m.reload()
}

// Watch starts a goroutine that watches the config file for changes.
func (m *Manager) Watch() {
	m.v.WatchConfig()
	m.v.OnConfigChange(func(_ fsnotify.Event) {
		if err := m.Reload(); err != nil {
			fmt.Fprintf(os.Stderr, "config reload failed: %v\n", err)
		}
	})
}

// reload unmarshals viper state into Config and applies defaults.
// Must be called with m.mu write-locked.
func (m *Manager) reload() error {
	var cfg Config
	if err := m.v.Unmarshal(&cfg); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}
	setDefaults(&cfg)
	m.current = &cfg
	for _, fn := range m.onChange {
		fn(&cfg)
	}
	return nil
}
