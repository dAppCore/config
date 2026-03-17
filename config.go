// Package config provides layered configuration management for the Core framework.
//
// Configuration values are resolved in priority order: defaults -> file -> env -> flags.
// Values are stored in a YAML file at ~/.core/config.yaml by default.
//
// Keys use dot notation for nested access:
//
//	cfg.Set("dev.editor", "vim")
//	var editor string
//	cfg.Get("dev.editor", &editor)
package config

import (
	"iter"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"

	coreio "forge.lthn.ai/core/go-io"
	coreerr "forge.lthn.ai/core/go-log"
	core "forge.lthn.ai/core/go/pkg/core"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// Config implements the core.Config interface with layered resolution.
// It uses viper as the underlying configuration engine.
type Config struct {
	mu     sync.RWMutex
	v      *viper.Viper // Full configuration (file + env + defaults)
	f      *viper.Viper // File-backed configuration only (for persistence)
	medium coreio.Medium
	path   string
}

// Option is a functional option for configuring a Config instance.
type Option func(*Config)

// WithMedium sets the storage medium for configuration file operations.
func WithMedium(m coreio.Medium) Option {
	return func(c *Config) {
		c.medium = m
	}
}

// WithPath sets the path to the configuration file.
func WithPath(path string) Option {
	return func(c *Config) {
		c.path = path
	}
}

// WithEnvPrefix sets the prefix for environment variables.
func WithEnvPrefix(prefix string) Option {
	return func(c *Config) {
		c.v.SetEnvPrefix(prefix)
	}
}

// New creates a new Config instance with the given options.
// If no medium is provided, it defaults to io.Local.
// If no path is provided, it defaults to ~/.core/config.yaml.
func New(opts ...Option) (*Config, error) {
	c := &Config{
		v: viper.New(),
		f: viper.New(),
	}

	// Configure viper defaults
	c.v.SetEnvPrefix("CORE_CONFIG")
	c.v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, opt := range opts {
		opt(c)
	}

	if c.medium == nil {
		c.medium = coreio.Local
	}

	if c.path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, coreerr.E("config.New", "failed to determine home directory", err)
		}
		c.path = filepath.Join(home, ".core", "config.yaml")
	}

	c.v.AutomaticEnv()

	// Load existing config file if it exists
	if c.medium.Exists(c.path) {
		if err := c.LoadFile(c.medium, c.path); err != nil {
			return nil, coreerr.E("config.New", "failed to load config file", err)
		}
	}

	return c, nil
}

// LoadFile reads a configuration file from the given medium and path and merges it into the current config.
// It supports YAML and environment files (.env).
func (c *Config) LoadFile(m coreio.Medium, path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	content, err := m.Read(path)
	if err != nil {
		return coreerr.E("config.LoadFile", "failed to read config file: "+path, err)
	}

	ext := filepath.Ext(path)
	configType := "yaml"
	if ext == "" && filepath.Base(path) == ".env" {
		configType = "env"
	} else if ext != "" {
		configType = strings.TrimPrefix(ext, ".")
	}

	// Load into file-backed viper
	c.f.SetConfigType(configType)
	if err := c.f.MergeConfig(strings.NewReader(content)); err != nil {
		return coreerr.E("config.LoadFile", "failed to parse config file (f): "+path, err)
	}

	// Load into full viper
	c.v.SetConfigType(configType)
	if err := c.v.MergeConfig(strings.NewReader(content)); err != nil {
		return coreerr.E("config.LoadFile", "failed to parse config file (v): "+path, err)
	}

	return nil
}

// Get retrieves a configuration value by dot-notation key and stores it in out.
// If key is empty, it unmarshals the entire configuration into out.
// The out parameter must be a pointer to the target type.
func (c *Config) Get(key string, out any) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if key == "" {
		if err := c.v.Unmarshal(out); err != nil {
			return coreerr.E("config.Get", "failed to unmarshal full config", err)
		}
		return nil
	}

	if !c.v.IsSet(key) {
		return coreerr.E("config.Get", "key not found: "+key, nil)
	}

	if err := c.v.UnmarshalKey(key, out); err != nil {
		return coreerr.E("config.Get", "failed to unmarshal key: "+key, err)
	}
	return nil
}

// Set stores a configuration value in memory.
// Call Commit() to persist changes to disk.
func (c *Config) Set(key string, v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.f.Set(key, v)
	c.v.Set(key, v)
	return nil
}

// Commit persists any changes made via Set() to the configuration file on disk.
// This will only save the configuration that was loaded from the file or explicitly Set(),
// preventing environment variable leakage.
func (c *Config) Commit() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := Save(c.medium, c.path, c.f.AllSettings()); err != nil {
		return coreerr.E("config.Commit", "failed to save config", err)
	}
	return nil
}

// All returns an iterator over all configuration values (including environment variables).
func (c *Config) All() iter.Seq2[string, any] {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return maps.All(c.v.AllSettings())
}

// Path returns the path to the configuration file.
func (c *Config) Path() string {
	return c.path
}

// Load reads a YAML configuration file from the given medium and path.
// Returns the parsed data as a map, or an error if the file cannot be read or parsed.
// Deprecated: Use Config.LoadFile instead.
func Load(m coreio.Medium, path string) (map[string]any, error) {
	content, err := m.Read(path)
	if err != nil {
		return nil, coreerr.E("config.Load", "failed to read config file: "+path, err)
	}

	v := viper.New()
	v.SetConfigType("yaml")
	if err := v.ReadConfig(strings.NewReader(content)); err != nil {
		return nil, coreerr.E("config.Load", "failed to parse config file: "+path, err)
	}

	return v.AllSettings(), nil
}

// Save writes configuration data to a YAML file at the given path.
// It ensures the parent directory exists before writing.
func Save(m coreio.Medium, path string, data map[string]any) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return coreerr.E("config.Save", "failed to marshal config", err)
	}

	dir := filepath.Dir(path)
	if err := m.EnsureDir(dir); err != nil {
		return coreerr.E("config.Save", "failed to create config directory: "+dir, err)
	}

	if err := m.Write(path, string(out)); err != nil {
		return coreerr.E("config.Save", "failed to write config file: "+path, err)
	}

	return nil
}

// Ensure Config implements core.Config at compile time.
var _ core.Config = (*Config)(nil)
