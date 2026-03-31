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
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sort"
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
	full   *viper.Viper // Full configuration (file + env + defaults)
	file   *viper.Viper // File-backed configuration only (for persistence)
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
		c.full.SetEnvPrefix(strings.TrimSuffix(prefix, "_"))
	}
}

// New creates a new Config instance with the given options.
// If no medium is provided, it defaults to io.Local.
// If no path is provided, it defaults to ~/.core/config.yaml.
func New(opts ...Option) (*Config, error) {
	c := &Config{
		full: viper.New(),
		file: viper.New(),
	}

	// Configure viper defaults
	c.full.SetEnvPrefix("CORE_CONFIG")
	c.full.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

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

	c.full.AutomaticEnv()

	// Load existing config file if it exists
	if c.medium.Exists(c.path) {
		if err := c.LoadFile(c.medium, c.path); err != nil {
			return nil, coreerr.E("config.New", "failed to load config file", err)
		}
	}

	return c, nil
}

func configTypeForPath(path string) (string, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" && filepath.Base(path) == ".env" {
		return "env", nil
	}
	if ext == "" {
		return "yaml", nil
	}

	switch ext {
	case ".yaml", ".yml":
		return "yaml", nil
	case ".json":
		return "json", nil
	case ".toml":
		return "toml", nil
	case ".env":
		return "env", nil
	default:
		return "", coreerr.E("config.configTypeForPath", "unsupported config file type: "+path, nil)
	}
}

// LoadFile reads a configuration file from the given medium and path and merges it into the current config.
// It supports YAML, JSON, TOML, and dotenv files (.env).
func (c *Config) LoadFile(m coreio.Medium, path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	configType, err := configTypeForPath(path)
	if err != nil {
		return coreerr.E("config.LoadFile", "failed to determine config file type: "+path, err)
	}

	content, err := m.Read(path)
	if err != nil {
		return coreerr.E("config.LoadFile", fmt.Sprintf("failed to read config file: %s", path), err)
	}

	parsed := viper.New()
	parsed.SetConfigType(configType)
	if err := parsed.MergeConfig(strings.NewReader(content)); err != nil {
		return coreerr.E("config.LoadFile", fmt.Sprintf("failed to parse config file: %s", path), err)
	}

	settings := parsed.AllSettings()

	// Keep the persisted and runtime views aligned with the same parsed data.
	if err := c.file.MergeConfigMap(settings); err != nil {
		return coreerr.E("config.LoadFile", "failed to merge config into file settings", err)
	}

	if err := c.full.MergeConfigMap(settings); err != nil {
		return coreerr.E("config.LoadFile", "failed to merge config into full settings", err)
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
		if err := c.full.Unmarshal(out); err != nil {
			return coreerr.E("config.Get", "failed to unmarshal full config", err)
		}
		return nil
	}

	if !c.full.IsSet(key) {
		return coreerr.E("config.Get", fmt.Sprintf("key not found: %s", key), nil)
	}

	if err := c.full.UnmarshalKey(key, out); err != nil {
		return coreerr.E("config.Get", fmt.Sprintf("failed to unmarshal key: %s", key), err)
	}
	return nil
}

// Set stores a configuration value in memory.
// Call Commit() to persist changes to disk.
func (c *Config) Set(key string, v any) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.file.Set(key, v)
	c.full.Set(key, v)
	return nil
}

// Commit persists any changes made via Set() to the configuration file on disk.
// This will only save the configuration that was loaded from the file or explicitly Set(),
// preventing environment variable leakage.
func (c *Config) Commit() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := Save(c.medium, c.path, c.file.AllSettings()); err != nil {
		return coreerr.E("config.Commit", "failed to save config", err)
	}
	return nil
}

// All returns an iterator over all configuration values in lexical key order
// (including environment variables).
func (c *Config) All() iter.Seq2[string, any] {
	c.mu.RLock()
	defer c.mu.RUnlock()

	settings := c.full.AllSettings()
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	return func(yield func(string, any) bool) {
		for _, key := range keys {
			if !yield(key, settings[key]) {
				return
			}
		}
	}
}

// Path returns the path to the configuration file.
func (c *Config) Path() string {
	return c.path
}

// Load reads a YAML configuration file from the given medium and path.
// Returns the parsed data as a map, or an error if the file cannot be read or parsed.
// Deprecated: Use Config.LoadFile instead.
func Load(m coreio.Medium, path string) (map[string]any, error) {
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case "", ".yaml", ".yml":
		// These paths are safe to treat as YAML sources.
	default:
		return nil, coreerr.E("config.Load", "unsupported config file type: "+path, nil)
	}

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
	switch ext := strings.ToLower(filepath.Ext(path)); ext {
	case "", ".yaml", ".yml":
		// These paths are safe to treat as YAML destinations.
	default:
		return coreerr.E("config.Save", "unsupported config file type: "+path, nil)
	}

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
