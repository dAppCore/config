// Package config provides layered configuration management for the Core framework.
package config

import (
	"fmt"
	"iter"
	"reflect"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"gopkg.in/yaml.v3"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"github.com/spf13/viper"
)

// Config provides runtime configuration with dual-viper storage.
//
// full stores values used for reads (file + env + explicit sets).
// file stores only values intended for persistence (file + explicit sets).
type Config struct {
	mu sync.RWMutex
	// full merges file data, environment variables, and local overrides.
	full *viper.Viper
	// file merges file data and local overrides only (no env).
	file   *viper.Viper
	medium coreio.Medium
	path   string
	env    string

	// Watch state.
	watchMu     sync.Mutex
	watcher     *fsnotify.Watcher
	watcherStop chan struct{}
	watcherDone chan struct{}
	watchTimer  *time.Timer
	watchers    []func(key string, value any)
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
//
// The prefix may include a trailing underscore; it is normalised for
// downstream viper configuration.
func WithEnvPrefix(prefix string) Option {
	return func(c *Config) {
		if prefix == "" {
			return
		}
		c.env = strings.TrimSuffix(prefix, "_")
	}
}

// New creates a new Config and loads values from the configured path if present.
//
// If no medium is provided, defaults to io.Local.
// If no path is provided, defaults to $DIR_HOME/.core/config.yaml.
func New(opts ...Option) (*Config, error) {
	return newConfig(true, opts...)
}

func defaultConfigPath() (string, error) {
	home := core.Env("DIR_HOME")
	if home == "" {
		return "", coreerr.E("config.New", "failed to resolve home directory", nil)
	}
	return core.Path(home, ".core", "config.yaml"), nil
}

func newConfig(loadFile bool, opts ...Option) (*Config, error) {
	c := &Config{
		full: viper.New(),
		file: viper.New(),
		env:  "CORE_CONFIG",
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.env == "" {
		c.env = "CORE_CONFIG"
	}
	if c.medium == nil {
		c.medium = coreio.Local
	}
	if c.path == "" {
		var err error
		c.path, err = defaultConfigPath()
		if err != nil {
			return nil, err
		}
	}

	c.full.SetEnvPrefix(c.env)
	c.full.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	c.full.AutomaticEnv()

	if loadFile && c.medium.Exists(c.path) {
		if err := c.LoadFile(c.medium, c.path); err != nil {
			return nil, err
		}
	}

	return c, nil
}

func configTypeForPath(path string) (string, error) {
	ext := core.Lower(core.PathExt(path))
	if ext == "" && core.PathBase(path) == ".env" {
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
//
// Supports YAML, JSON, TOML, and dotenv files.
func (c *Config) LoadFile(m coreio.Medium, path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	configType, err := configTypeForPath(path)
	if err != nil {
		return coreerr.E("config.LoadFile", "failed to determine config file type: "+path, err)
	}

	content, err := m.Read(path)
	if err != nil {
		return core.E("config.LoadFile", "failed to read config file: "+path, err)
	}

	parsed := viper.New()
	parsed.SetConfigType(configType)
	if err := parsed.MergeConfig(core.NewReader(content)); err != nil {
		return coreerr.E("config.LoadFile", core.Sprintf("failed to parse config file: %s", path), err)
	}

	settings := parsed.AllSettings()

	if err := c.file.MergeConfigMap(settings); err != nil {
		return coreerr.E("config.LoadFile", "failed to merge config into file settings", err)
	}

	if err := c.full.MergeConfigMap(settings); err != nil {
		return coreerr.E("config.LoadFile", "failed to merge config into full settings", err)
	}

	return nil
}

// Get retrieves a configuration value by dot-notation key and stores it in out.
//
// If key is empty, unmarshal the full configuration into out.
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
		return coreerr.E("config.Get", core.Sprintf("key not found: %s", key), nil)
	}

	if err := c.full.UnmarshalKey(key, out); err != nil {
		return coreerr.E("config.Get", core.Sprintf("failed to unmarshal key: %s", key), err)
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

// Commit persists in-memory updates to disk.
// Environment variables are never written.
func (c *Config) Commit() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := Save(c.medium, c.path, c.file.AllSettings()); err != nil {
		return coreerr.E("config.Commit", "failed to save config", err)
	}
	return nil
}

// MergeFrom overlays source values onto the receiver.
// Existing keys in the receiver are not overwritten.
func (c *Config) MergeFrom(source *Config) {
	if source == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	source.mu.RLock()
	sourceFull := cloneSettings(source.full.AllSettings())
	sourceFile := cloneSettings(source.file.AllSettings())
	source.mu.RUnlock()

	mergeMissingMaps(c.full.AllSettings(), sourceFull)
	mergeMissingMaps(c.file.AllSettings(), sourceFile)
}

// Load reads a YAML or `.env` config file.
//
// Returns the parsed data as a map, or an error if the file cannot be read or parsed.
//
// Deprecated: Use Config.LoadFile instead.
func Load(m coreio.Medium, path string) (map[string]any, error) {
	configType, err := yamlConfigTypeForPath(path)
	if err != nil {
		return nil, coreerr.E("config.Load", err.Error(), nil)
	}

	content, err := m.Read(path)
	if err != nil {
		return nil, core.E("config.Load", "failed to read config file: "+path, err)
	}

	v := viper.New()
	v.SetConfigType(configType)
	if err := v.ReadConfig(core.NewReader(content)); err != nil {
		return nil, coreerr.E("config.Load", "failed to parse config file: "+path, err)
	}

	return v.AllSettings(), nil
}

// yamlConfigTypeForPath restricts Load to YAML and `.env` sources.
//
// The full multi-format parser lives in configTypeForPath for use by LoadFile.
func yamlConfigTypeForPath(path string) (string, error) {
	ext := core.Lower(core.PathExt(path))
	if ext == "" && core.PathBase(path) == ".env" {
		return "env", nil
	}
	if ext == "" {
		return "yaml", nil
	}

	switch ext {
	case ".yaml", ".yml":
		return "yaml", nil
	case ".env":
		return "env", nil
	default:
		return "", coreerr.E("config.yamlConfigTypeForPath", "unsupported config file type: "+path, nil)
	}
}

// Save writes configuration data to a YAML file at the given path.
// It ensures the parent directory exists before writing.
func Save(m coreio.Medium, path string, data map[string]any) error {
	switch ext := core.Lower(core.PathExt(path)); ext {
	case "", ".yaml", ".yml":
	default:
		return coreerr.E("config.Save", "unsupported config file type: "+path, nil)
	}

	out, err := yaml.Marshal(data)
	if err != nil {
		return core.E("config.Save", "failed to marshal config", err)
	}

	dir := core.PathDir(path)
	if err := m.EnsureDir(dir); err != nil {
		return core.E("config.Save", "failed to create config directory: "+dir, err)
	}

	if err := m.Write(path, string(out)); err != nil {
		return core.E("config.Save", "failed to write config file: "+path, err)
	}

	return nil
}

// All returns an iterator over all configuration values in lexical key order.
func (c *Config) All() iter.Seq2[string, any] {
	c.mu.RLock()
	defer c.mu.RUnlock()

	settings := c.full.AllSettings()
	keys := make([]string, 0, len(settings))
	for key := range settings {
		keys = append(keys, key)
	}
	slices.Sort(keys)

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

func cloneSettings(input map[string]any) map[string]any {
	if len(input) == 0 {
		return map[string]any{}
	}
	copyAll := make(map[string]any, len(input))
	for k, v := range input {
		copyAll[k] = cloneValue(v)
	}
	return copyAll
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneSettings(typed)
	case []any:
		vals := make([]any, len(typed))
		for i, v := range typed {
			vals[i] = cloneValue(v)
		}
		return vals
	default:
		return value
	}
}

func mergeMissingMaps(target map[string]any, source map[string]any) {
	for key, value := range source {
		existing, ok := target[key]
		if !ok {
			target[key] = value
			continue
		}

		srcMap, srcIsMap := toStringMap(value)
		tgtMap, tgtIsMap := toStringMap(existing)
		if srcIsMap && tgtIsMap {
			mergeMissingMaps(tgtMap, srcMap)
		}
	}
}

func toStringMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func flattenForDiff(prefix string, data any, out map[string]any) {
	switch typed := data.(type) {
	case map[string]any:
		if len(typed) == 0 && prefix != "" {
			out[prefix] = map[string]any{}
			return
		}
		for key, value := range typed {
			next := key
			if prefix != "" {
				next = fmt.Sprintf("%s.%s", prefix, key)
			}
			flattenForDiff(next, value, out)
		}
	case []any:
		out[prefix] = cloneValue(typed)
	default:
		out[prefix] = typed
	}
}

func changedKeys(old map[string]any, latest map[string]any) map[string]any {
	result := make(map[string]any)
	all := make(map[string]struct{}, len(old)+len(latest))
	for key := range old {
		all[key] = struct{}{}
	}
	for key := range latest {
		all[key] = struct{}{}
	}

	for key := range all {
		oldVal, oldOK := old[key]
		newVal, newOK := latest[key]

		if !oldOK {
			result[key] = newVal
			continue
		}
		if !newOK {
			result[key] = nil
			continue
		}
		if !reflect.DeepEqual(oldVal, newVal) {
			result[key] = newVal
		}
	}

	return result
}

func (c *Config) emitChanges(changed map[string]any) {
	if len(changed) == 0 {
		return
	}

	keys := make([]string, 0, len(changed))
	for key := range changed {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	c.watchMu.Lock()
	callbacks := append([]func(string, any){}, c.watchers...)
	c.watchMu.Unlock()

	for _, key := range keys {
		for _, fn := range callbacks {
			fn(key, changed[key])
		}
	}
}
