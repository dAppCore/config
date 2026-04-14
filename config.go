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
	"fmt"
	"iter"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

// ConfigChanged is broadcast on every Set() and Commit() call so other services
// can react to runtime config updates without polling.
//
//	c.RegisterAction(func(c *core.Core, msg core.Message) core.Result {
//	    if cc, ok := msg.(config.ConfigChanged); ok {
//	        // react to cc.Key / cc.Value
//	    }
//	    return core.Result{}
//	})
type ConfigChanged struct {
	Key      string
	Value    any
	Previous any
	Source   string // "set", "env", "file", "commit"
}

// Config implements layered configuration with a dual-Viper pattern.
// The full viper holds file + env + defaults (for reads); the file viper
// holds file + explicit Set() calls only (for persistence).
//
//	cfg, _ := config.New(config.WithPath("~/.core/config.yaml"))
//	cfg.Set("dev.editor", "vim")
//	cfg.Commit()
type Config struct {
	mu        sync.RWMutex
	full      *viper.Viper // Full configuration (file + env + defaults)
	file      *viper.Viper // File-backed configuration only (for persistence)
	medium    coreio.Medium
	path      string
	core      *core.Core // optional — set when attached to a Core service
	callbacks []func(key string, value any)
	watcher   *fileWatcher
}

// Option is a functional option for configuring a Config instance.
type Option func(*Config)

// WithMedium sets the storage medium for configuration file operations.
//
//	config.New(config.WithMedium(io.Local))
func WithMedium(m coreio.Medium) Option {
	return func(c *Config) {
		c.medium = m
	}
}

// WithPath sets the path to the configuration file.
//
//	config.New(config.WithPath("~/.core/config.yaml"))
func WithPath(path string) Option {
	return func(c *Config) {
		c.path = path
	}
}

// WithEnvPrefix sets the prefix for environment variables.
//
//	config.New(config.WithEnvPrefix("CORE_CONFIG"))  // CORE_CONFIG_DEV_EDITOR → dev.editor
func WithEnvPrefix(prefix string) Option {
	return func(c *Config) {
		c.full.SetEnvPrefix(strings.TrimSuffix(prefix, "_"))
	}
}

// WithCore attaches the Config to a Core instance so Set()/Commit() calls
// broadcast ConfigChanged events.
//
//	config.New(config.WithCore(c))
func WithCore(c *core.Core) Option {
	return func(cfg *Config) {
		cfg.core = c
	}
}

// AttachCore wires the Config to a Core instance after construction. Use this
// when New() ran before Core was available (e.g. from a service lifecycle).
// Thread-safe; safe to call concurrently with Set()/Commit().
//
//	cfg, _ := config.New(...)
//	cfg.AttachCore(c)       // ConfigChanged broadcasts from here on.
func (c *Config) AttachCore(core *core.Core) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.core = core
}

// New creates a new Config instance with the given options.
// If no medium is provided, it defaults to io.Local.
// If no path is provided, it defaults to ~/.core/config.yaml.
//
//	cfg, err := config.New(
//	    config.WithPath("~/.core/config.yaml"),
//	    config.WithMedium(io.Local),
//	    config.WithEnvPrefix("CORE_CONFIG"),
//	)
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

// LoadFile reads a configuration file from the given medium and path and merges it
// into the current config. It supports YAML, JSON, TOML, and dotenv files (.env).
//
//	cfg.LoadFile(io.Local, ".core/build.yaml")
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
//
//	var editor string
//	cfg.Get("dev.editor", &editor)
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

// Set stores a configuration value in memory and broadcasts ConfigChanged.
// Call Commit() to persist changes to disk.
//
//	cfg.Set("dev.editor", "vim")
//	cfg.Commit()
func (c *Config) Set(key string, v any) error {
	c.mu.Lock()
	previous := c.full.Get(key)
	c.file.Set(key, v)
	c.full.Set(key, v)
	callbacks := append([]func(string, any){}, c.callbacks...)
	attached := c.core
	c.mu.Unlock()

	for _, fn := range callbacks {
		fn(key, v)
	}
	if attached != nil {
		attached.ACTION(ConfigChanged{Key: key, Value: v, Previous: previous, Source: "set"})
	}
	return nil
}

// Commit persists any changes made via Set() to the configuration file on disk.
// This will only save the configuration that was loaded from the file or explicitly Set(),
// preventing environment variable leakage.
//
//	cfg.Commit()
func (c *Config) Commit() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := Save(c.medium, c.path, c.file.AllSettings()); err != nil {
		return coreerr.E("config.Commit", "failed to save config", err)
	}
	if c.core != nil {
		c.core.ACTION(ConfigChanged{Key: "", Value: nil, Source: "commit"})
	}
	return nil
}

// All returns an iterator over every configuration value in lexical key order.
// Keys are flat dot-notation (e.g. "dev.editor", "app.name"). The iterator
// includes values sourced from file, Set() calls, and environment-variable
// overrides mapped via the configured env prefix (so CORE_CONFIG_DEV_EDITOR
// shows up as "dev.editor").
//
//	for key, value := range cfg.All() {
//	    fmt.Println(key, value)   // "dev.editor" "vim"
//	}
func (c *Config) All() iter.Seq2[string, any] {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// AllKeys() gives flat dot-notation for every key viper knows about,
	// covering file values, explicit Set() calls, and registered env overrides.
	keys := append([]string(nil), c.full.AllKeys()...)

	// Surface env-prefixed variables that were never declared in the file so
	// the iterator truly reflects the merged reality rather than only the
	// persisted surface.
	prefix := envPrefixOf(c.full)
	if prefix != "" {
		for envKey := range Env(prefix) {
			if !contains(keys, envKey) {
				keys = append(keys, envKey)
			}
		}
	}

	sort.Strings(keys)

	return func(yield func(string, any) bool) {
		for _, key := range keys {
			if !yield(key, c.full.Get(key)) {
				return
			}
		}
	}
}

// envPrefixOf returns the environment-variable prefix registered with viper
// in the form required by Env() (trailing underscore, uppercase). Empty
// when no prefix is active.
func envPrefixOf(v *viper.Viper) string {
	// Viper stores the prefix verbatim (without trailing underscore) via
	// SetEnvPrefix. We probe via its canonical environment-key transform.
	canonical := v.GetEnvPrefix()
	if canonical == "" {
		return ""
	}
	if !strings.HasSuffix(canonical, "_") {
		canonical += "_"
	}
	return canonical
}

// Path returns the path to the configuration file.
//
//	cfg.Path()  // "~/.core/config.yaml"
func (c *Config) Path() string {
	return c.path
}

// Medium returns the I/O medium backing the configuration.
//
//	medium := cfg.Medium()
func (c *Config) Medium() coreio.Medium {
	return c.medium
}

// MergeFrom overlays source values onto the receiver at the leaf level.
// Existing keys in the receiver are NOT overwritten — including keys sourced
// from environment variables via AutomaticEnv. Used by Discover() to merge
// closer (project) configs over further (global) ones without shadowing the
// "env overrides everything" rule from §5.3 of the .core/ convention spec.
//
// Inherited values land in the defaults layer only — Commit() never persists
// them into this Config's own file, so a project Commit cannot leak secrets
// discovered from ~/.core/config.yaml into the project config.
//
//	base := config.New()
//	base.MergeFrom(projectConfig)   // closest wins
//	base.MergeFrom(globalConfig)    // fills gaps only
func (c *Config) MergeFrom(source *Config) {
	if source == nil {
		return
	}
	source.mu.RLock()
	// file keys are the source's own persistable values. full keys also
	// include values previously inherited from other MergeFrom calls — we
	// need both so inheritance chains (discover → conclave) keep working.
	leaf := map[string]any{}
	for _, key := range source.file.AllKeys() {
		leaf[key] = source.file.Get(key)
	}
	for _, key := range source.full.AllKeys() {
		if _, ok := leaf[key]; ok {
			continue
		}
		leaf[key] = source.full.Get(key)
	}
	source.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()
	for key, value := range leaf {
		if c.full.IsSet(key) {
			continue
		}
		// full viper uses SetDefault so AutomaticEnv still wins on Get, but
		// file viper is left untouched — inherited values must not be written
		// back when Commit() flushes this Config to disk.
		c.full.SetDefault(key, value)
	}
}

// OnChange registers a callback invoked when a config key changes.
// The callback receives the key path and new value.
// Multiple callbacks can be registered; all are called in order.
//
//	cfg.OnChange(func(key string, value any) {
//	    if key == "dev.editor" {
//	        fmt.Println("editor changed to", value)
//	    }
//	})
func (c *Config) OnChange(fn func(key string, value any)) {
	if fn == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.callbacks = append(c.callbacks, fn)
}

// Load reads a YAML configuration file from the given medium and path.
// Returns the parsed data as a map, or an error if the file cannot be read or parsed.
//
//	data, err := config.Load(io.Local, "~/.core/config.yaml")
//
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
//
//	config.Save(io.Local, "~/.core/config.yaml", map[string]any{"dev": map[string]any{"editor": "vim"}})
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
