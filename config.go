// Package config provides layered configuration management for the Core framework.
//
// Configuration values are resolved in priority order: defaults -> file -> env -> Set().
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
	"sort"
	"sync"

	core "dappco.re/go"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

const (
	errConfigIsNil               = "config is nil"
	errStorageMediumIsNil        = "storage medium is nil"
	errUnsupportedConfigFileType = "unsupported config file type: "
	operationConfigGet           = "config.Get"
	operationConfigLoad          = "config.Load"
	operationConfigLoadFile      = "config.LoadFile"
	operationConfigSave          = "config.Save"
)

// Medium is the storage surface Config needs for file-backed settings.
type Medium interface {
	Exists(path string) bool
	Read(path string) core.Result
	Write(path, content string) core.Result
	EnsureDir(path string) core.Result
}

// Config implements the core.Config interface with layered resolution.
// It uses viper as the underlying configuration engine.
type Config struct {
	mu        sync.RWMutex
	full      *viper.Viper // Full configuration (file + env + defaults)
	file      *viper.Viper // File-backed configuration only (for persistence)
	medium    Medium
	path      string
	envPrefix string
	overrides map[string]any
}

// Option is a functional option for configuring a Config instance.
type Option func(*Config)

// WithMedium sets the storage medium for configuration file operations.
func WithMedium(m Medium) Option {
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
		c.envPrefix = core.TrimSuffix(prefix, "_")
	}
}

// New creates a new Config instance with the given options.
// If no medium is provided, it defaults to io.Local.
// If no path is provided, it defaults to ~/.core/config.yaml.
func New(opts ...Option) core.Result {
	c := &Config{
		full:      viper.New(),
		file:      viper.New(),
		envPrefix: "CORE_CONFIG",
		overrides: make(map[string]any),
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.medium == nil {
		c.medium = (&core.Fs{}).New("/")
	}

	if c.path == "" {
		home := core.UserHomeDir()
		if !home.OK {
			return core.Fail(core.E("config.New", "failed to determine home directory", core.NewError(home.Error())))
		}
		c.path = core.Path(home.Value.(string), ".core", "config.yaml")
	}

	// Load existing config file if it exists
	if c.medium.Exists(c.path) {
		if r := c.LoadFile(c.medium, c.path); !r.OK {
			return core.Fail(core.E("config.New", "failed to load config file", core.NewError(r.Error())))
		}
	} else if r := c.refreshFull(); !r.OK {
		return r
	}

	return core.Ok(c)
}

func configTypeForPath(path string) core.Result {
	ext := core.Lower(core.PathExt(path))
	if ext == "" && core.PathBase(path) == ".env" {
		return core.Ok("env")
	}
	if ext == "" {
		return core.Ok("yaml")
	}

	switch ext {
	case ".yaml", ".yml":
		return core.Ok("yaml")
	case ".json":
		return core.Ok("json")
	case ".toml":
		return core.Ok("toml")
	case ".env":
		return core.Ok("env")
	default:
		return core.Fail(core.E("config.configTypeForPath", errUnsupportedConfigFileType+path, nil))
	}
}

func (c *Config) refreshFull() core.Result {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.refreshFullLocked()
}

func (c *Config) refreshFullLocked() core.Result {
	next := viper.New()
	if r := core.ResultOf(nil, next.MergeConfigMap(c.file.AllSettings())); !r.OK {
		return core.Fail(core.E("config.refreshFull", "failed to merge file settings", core.NewError(r.Error())))
	}
	for key, value := range Env(c.envPrefix) {
		next.Set(key, value)
	}
	for key, value := range c.overrides {
		next.Set(key, value)
	}
	c.full = next
	return core.Ok(nil)
}

// LoadFile reads a configuration file from the given medium and path and merges it into the current config.
// It supports YAML, JSON, TOML, and dotenv files (.env).
func (c *Config) LoadFile(m Medium, path string) core.Result {
	if c == nil {
		return core.Fail(core.E(operationConfigLoadFile, errConfigIsNil, nil))
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if m == nil {
		return core.Fail(core.E(operationConfigLoadFile, errStorageMediumIsNil, nil))
	}

	configType := configTypeForPath(path)
	if !configType.OK {
		return core.Fail(core.E(operationConfigLoadFile, "failed to determine config file type: "+path, core.NewError(configType.Error())))
	}

	read := m.Read(path)
	if !read.OK {
		return core.Fail(core.E(operationConfigLoadFile, core.Sprintf("failed to read config file: %s", path), core.NewError(read.Error())))
	}

	content, ok := read.Value.(string)
	if !ok {
		return core.Fail(core.E(operationConfigLoadFile, core.Sprintf("config file was not text: %s", path), nil))
	}

	parsed := viper.New()
	parsed.SetConfigType(configType.Value.(string))
	if r := core.ResultOf(nil, parsed.MergeConfig(core.NewReader(content))); !r.OK {
		return core.Fail(core.E(operationConfigLoadFile, core.Sprintf("failed to parse config file: %s", path), core.NewError(r.Error())))
	}

	settings := parsed.AllSettings()

	// Keep the persisted and runtime views aligned with the same parsed data.
	if r := core.ResultOf(nil, c.file.MergeConfigMap(settings)); !r.OK {
		return core.Fail(core.E(operationConfigLoadFile, "failed to merge config into file settings", core.NewError(r.Error())))
	}

	return c.refreshFullLocked()
}

// Get retrieves a configuration value by dot-notation key and stores it in out.
// If key is empty, it unmarshals the entire configuration into out.
// The out parameter must be a pointer to the target type.
func (c *Config) Get(key string, out any) core.Result {
	if c == nil {
		return core.Fail(core.E(operationConfigGet, errConfigIsNil, nil))
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if r := c.refreshFullLocked(); !r.OK {
		return r
	}

	if key == "" {
		if r := core.ResultOf(nil, c.full.Unmarshal(out)); !r.OK {
			return core.Fail(core.E(operationConfigGet, "failed to unmarshal full config", core.NewError(r.Error())))
		}
		return core.Ok(out)
	}

	if !c.full.IsSet(key) {
		return core.Fail(core.E(operationConfigGet, core.Sprintf("key not found: %s", key), nil))
	}

	if r := core.ResultOf(nil, c.full.UnmarshalKey(key, out)); !r.OK {
		return core.Fail(core.E(operationConfigGet, core.Sprintf("failed to unmarshal key: %s", key), core.NewError(r.Error())))
	}
	return core.Ok(out)
}

// Set stores a configuration value in memory.
// Call Commit() to persist changes to disk.
func (c *Config) Set(key string, v any) core.Result {
	if c == nil {
		return core.Fail(core.E("config.Set", errConfigIsNil, nil))
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	c.file.Set(key, v)
	c.overrides[key] = v
	return c.refreshFullLocked()
}

// Commit persists any changes made via Set() to the configuration file on disk.
// This will only save the configuration that was loaded from the file or explicitly Set(),
// preventing environment variable leakage.
func (c *Config) Commit() core.Result {
	if c == nil {
		return core.Fail(core.E("config.Commit", errConfigIsNil, nil))
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	if r := Save(c.medium, c.path, c.file.AllSettings()); !r.OK {
		return core.Fail(core.E("config.Commit", "failed to save config", core.NewError(r.Error())))
	}
	return core.Ok(nil)
}

// All returns an iterator over all configuration values in lexical key order
// (including environment variables).
func (c *Config) All() iter.Seq2[string, any] {
	c.mu.Lock()
	defer c.mu.Unlock()

	if r := c.refreshFullLocked(); !r.OK {
		return func(func(string, any) bool) {}
	}

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
func Load(m Medium, path string) core.Result {
	switch ext := core.Lower(core.PathExt(path)); ext {
	case "", ".yaml", ".yml":
		// These paths are safe to treat as YAML sources.
	default:
		return core.Fail(core.E(operationConfigLoad, errUnsupportedConfigFileType+path, nil))
	}

	if m == nil {
		return core.Fail(core.E(operationConfigLoad, errStorageMediumIsNil, nil))
	}

	read := m.Read(path)
	if !read.OK {
		return core.Fail(core.E(operationConfigLoad, "failed to read config file: "+path, core.NewError(read.Error())))
	}

	content, ok := read.Value.(string)
	if !ok {
		return core.Fail(core.E(operationConfigLoad, "config file was not text: "+path, nil))
	}

	v := viper.New()
	v.SetConfigType("yaml")
	if r := core.ResultOf(nil, v.ReadConfig(core.NewReader(content))); !r.OK {
		return core.Fail(core.E(operationConfigLoad, "failed to parse config file: "+path, core.NewError(r.Error())))
	}

	return core.Ok(v.AllSettings())
}

// Save writes configuration data to a YAML file at the given path.
// It ensures the parent directory exists before writing.
func Save(m Medium, path string, data map[string]any) core.Result {
	switch ext := core.Lower(core.PathExt(path)); ext {
	case "", ".yaml", ".yml":
		// These paths are safe to treat as YAML destinations.
	default:
		return core.Fail(core.E(operationConfigSave, errUnsupportedConfigFileType+path, nil))
	}

	if m == nil {
		return core.Fail(core.E(operationConfigSave, errStorageMediumIsNil, nil))
	}

	out := core.ResultOf(yaml.Marshal(data))
	if !out.OK {
		return core.Fail(core.E(operationConfigSave, "failed to marshal config", core.NewError(out.Error())))
	}

	dir := core.PathDir(path)
	if r := m.EnsureDir(dir); !r.OK {
		return core.Fail(core.E(operationConfigSave, "failed to create config directory: "+dir, core.NewError(r.Error())))
	}

	if r := m.Write(path, string(out.Value.([]byte))); !r.OK {
		return core.Fail(core.E(operationConfigSave, "failed to write config file: "+path, core.NewError(r.Error())))
	}

	return core.Ok(nil)
}
