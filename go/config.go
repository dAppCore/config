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
	"maps"
	"slices"
	"sync"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

type envkeyreplacer struct{}

func (envkeyreplacer) Replace(s string) string {
	return core.Replace(s, ".", "_")
}

const (
	callerConfigNew           = "config.New"
	callerConfigLoad          = "config.Load"
	callerConfigLoadFile      = "config.LoadFile"
	callerConfigGet           = "config.Get"
	callerConfigSave          = "config.Save"
	unsupportedConfigFileType = "unsupported config file type"
	configChangeSourceFile    = "file"
	configChangeSourceSet     = "set"
	configChangeSourceCommit  = "commit"
)

// ConfigChanged is broadcast on every Set() and Commit() call so other services
// can react to runtime config updates without polling.
//
//	c.RegisterAction(func(c *core.Core, msg core.Message) core.Result {
//	    if cc, ok := msg.(config.ConfigChanged); ok {
//	        // react to cc.Key / cc.Value
//	    }
//	    return core.Ok(nil)
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
	store     ConfigStoreWriter
	callbacks []func(key string, value any)
	watcher   *fileWatcher
}

// Option is a functional option for configuring a Config instance.
type Option func(*Config)

// ConfigStoreWriter is the minimal store contract config needs for mirroring
// Set() calls into go-store when available.
//
//	store.Set("config", "dev.editor", "\"vim\"")
type ConfigStoreWriter interface {
	Set(string, string, string) error
}

// ConfigStoreReader is the optional store contract for hydrating config state
// back into memory on startup so pre-Commit Set() calls survive restarts.
//
//	values, _ := store.GetAll("config")
type ConfigStoreReader interface {
	GetAll(string) (map[string]string, error)
}

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
		c.full.SetEnvPrefix(core.TrimSuffix(prefix, "_"))
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

// WithStore attaches an optional go-store-compatible writer. When present,
// every Set() call mirrors the new value into the "config" bucket.
//
//	config.New(config.WithStore(store))
func WithStore(store ConfigStoreWriter) Option {
	return func(c *Config) {
		c.store = store
	}
}

// WithDefaults seeds default values into the defaults layer, the lowest rung
// of the resolution priority (defaults → file → env → Set()). Existing file
// or env values are NOT shadowed — defaults only fill in gaps the caller has
// not supplied another way.
//
//	cfg, _ := config.New(config.WithDefaults(map[string]any{
//	    "dev.editor":  "vim",
//	    "app.version": "0.1.0",
//	}))
//	cfg.Get("dev.editor", &editor)  // "vim" unless file/env/Set overrides
func WithDefaults(defaults map[string]any) Option {
	return func(c *Config) {
		for key, value := range defaults {
			c.full.SetDefault(key, value)
		}
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
func New(opts ...Option) core.Result {
	return newConfig(true, opts...)
}

// newConfig centralises Config construction so discovery can create a config
// shell without eagerly loading the path that will later receive merged layers.
func newConfig(loadFromPath bool, opts ...Option) core.Result {
	c := &Config{
		full: viper.NewWithOptions(viper.EnvKeyReplacer(envkeyreplacer{})),
		file: viper.New(),
	}

	// Configure viper defaults
	c.full.SetEnvPrefix("CORE_CONFIG")

	for _, opt := range opts {
		opt(c)
	}

	if c.medium == nil {
		c.medium = coreio.Local
	}

	if c.path == "" {
		home := core.Env("DIR_HOME")
		if home == "" {
			return core.Fail(core.E(callerConfigNew, "failed to determine home directory", nil))
		}
		c.path = core.Path(home, ".core", "config.yaml")
	}

	c.full.AutomaticEnv()

	// Load existing config file if it exists.
	if loadFromPath && c.medium.Exists(c.path) {
		if r := c.loadFile(c.medium, c.path, false); !r.OK {
			return core.Fail(core.E(callerConfigNew, "failed to load config file", resultCause(r).(error)))
		}
	}
	if loadFromPath {
		if r := c.loadStoreState(); !r.OK {
			return core.Fail(core.E(callerConfigNew, "failed to load config store state", resultCause(r).(error)))
		}
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
		return core.Fail(core.E("config.configTypeForPath", "unsupported config file type: "+path, nil))
	}
}

// LoadFile reads a configuration file from the given medium and path and merges it
// into the current config. It supports YAML, JSON, TOML, and dotenv files (.env).
//
//	cfg.LoadFile(io.Local, ".core/build.yaml")
func (c *Config) LoadFile(m coreio.Medium, path string) core.Result {
	return c.loadFile(m, path, true)
}

// loadFile merges a configuration file into the current Config. When notify is
// true it also broadcasts ConfigChanged events for each changed key.
func (c *Config) loadFile(m coreio.Medium, path string, notify bool) core.Result {
	c.mu.Lock()
	before := c.snapshotAllLocked()

	settingsResult := readConfigSettings(m, path)
	if !settingsResult.OK {
		c.mu.Unlock()
		return settingsResult
	}
	settings := settingsResult.Value.(map[string]any)
	if r := c.mergeConfigSettingsLocked(settings); !r.OK {
		c.mu.Unlock()
		return r
	}

	callbacks, attached, after := c.loadNotificationStateLocked(notify)
	c.mu.Unlock()

	if notify && (len(callbacks) > 0 || attached != nil) {
		emitConfigChanges(callbacks, attached, diffSnapshots(before, after), configChangeSourceFile)
	}

	return core.Ok(nil)
}

func readConfigSettings(m coreio.Medium, path string) core.Result {
	configTypeResult := configTypeForPath(path)
	if !configTypeResult.OK {
		return core.Fail(core.E(callerConfigLoadFile, "failed to determine config file type: "+path, resultCause(configTypeResult).(error)))
	}
	configType := configTypeResult.Value.(string)

	content, err := m.Read(path)
	if err != nil {
		return core.Fail(core.E(callerConfigLoadFile, "failed to read config file: "+path, err))
	}

	parsed := viper.New()
	parsed.SetConfigType(configType)
	if err := parsed.MergeConfig(core.NewReader(content)); err != nil {
		return core.Fail(core.E(callerConfigLoadFile, core.Sprintf("failed to parse config file: %s", path), err))
	}

	settings := parsed.AllSettings()
	if r := validateSchema(path, settings); !r.OK {
		return r
	}
	return core.Ok(settings)
}

func (c *Config) mergeConfigSettingsLocked(settings map[string]any) core.Result {
	if err := c.file.MergeConfigMap(settings); err != nil {
		return core.Fail(core.E(callerConfigLoadFile, "failed to merge config into file settings", err))
	}
	if err := c.full.MergeConfigMap(settings); err != nil {
		return core.Fail(core.E(callerConfigLoadFile, "failed to merge config into full settings", err))
	}
	return core.Ok(nil)
}

func (c *Config) loadNotificationStateLocked(notify bool) ([]func(string, any), *core.Core, map[string]any) {
	callbacks := append([]func(string, any){}, c.callbacks...)
	attached := c.core
	if !notify || (len(callbacks) == 0 && attached == nil) {
		return callbacks, attached, nil
	}
	return callbacks, attached, c.snapshotAllLocked()
}

func emitConfigChanges(callbacks []func(string, any), attached *core.Core, changes []configChange, source string) {
	for _, change := range changes {
		for _, fn := range callbacks {
			fn(change.Key, change.Value)
		}
		if attached != nil {
			reportConfigBroadcast(attached.ACTION(ConfigChanged{
				Key:      change.Key,
				Value:    change.Value,
				Previous: change.Previous,
				Source:   source,
			}), source)
		}
	}
}

func reportConfigBroadcast(r core.Result, source string) {
	if r.OK {
		return
	}
	core.Warn("config change broadcast failed", "source", source, "err", r.Error())
}

// Get retrieves a configuration value by dot-notation key and stores it in out.
// If key is empty, it unmarshals the entire configuration into out.
// The out parameter must be a pointer to the target type.
//
//	var editor string
//	cfg.Get("dev.editor", &editor)
func (c *Config) Get(key string, out any) core.Result {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if key == "" {
		if err := c.full.Unmarshal(out); err != nil {
			return core.Fail(core.E(callerConfigGet, "failed to unmarshal full config", err))
		}
		return core.Ok(nil)
	}

	if !c.full.IsSet(key) {
		return core.Fail(core.E(callerConfigGet, core.Sprintf("key not found: %s", key), nil))
	}

	if err := c.full.UnmarshalKey(key, out); err != nil {
		return core.Fail(core.E(callerConfigGet, core.Sprintf("failed to unmarshal key: %s", key), err))
	}
	return core.Ok(nil)
}

// SetDefault stores a value in the lowest-precedence defaults layer. File,
// env and explicit Set() values all shadow defaults. Unlike Set(), defaults
// are NOT persisted by Commit() and do NOT broadcast ConfigChanged — they
// exist so callers can declare a baseline the config resolves to when no
// other source has spoken.
//
//	cfg.SetDefault("dev.editor", "vim")
//	cfg.Get("dev.editor", &editor)  // "vim" until someone Sets/loads another value
func (c *Config) SetDefault(key string, v any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.full.SetDefault(key, v)
}

// Set stores a configuration value in memory and broadcasts ConfigChanged.
// Call Commit() to persist changes to disk.
//
//	cfg.Set("dev.editor", "vim")
//	cfg.Commit()
func (c *Config) Set(key string, v any) core.Result {
	c.mu.Lock()
	previous := c.full.Get(key)
	c.file.Set(key, v)
	c.full.Set(key, v)
	store := c.store
	callbacks := append([]func(string, any){}, c.callbacks...)
	attached := c.core
	c.mu.Unlock()

	for _, fn := range callbacks {
		fn(key, v)
	}
	if attached != nil {
		reportConfigBroadcast(attached.ACTION(ConfigChanged{Key: key, Value: v, Previous: previous, Source: configChangeSourceSet}), configChangeSourceSet)
	}
	persistToStore(store, key, v)
	return core.Ok(nil)
}

// Commit persists any changes made via Set() to the configuration file on disk.
// This will only save the configuration that was loaded from the file or explicitly Set(),
// preventing environment variable leakage.
//
//	cfg.Commit()
func (c *Config) Commit() core.Result {
	c.mu.Lock()
	medium := c.medium
	path := c.path
	settings := c.file.AllSettings()
	attached := c.core
	c.mu.Unlock()

	if r := Save(medium, path, settings); !r.OK {
		return core.Fail(core.E("config.Commit", "failed to save config", resultCause(r).(error)))
	}
	if attached != nil {
		reportConfigBroadcast(attached.ACTION(ConfigChanged{Key: "", Value: nil, Source: configChangeSourceCommit}), configChangeSourceCommit)
	}
	return core.Ok(nil)
}

// All returns an iterator over a snapshot of every configuration value in
// lexical key order. Keys are flat dot-notation (e.g. "dev.editor",
// "app.name"). The iterator includes values sourced from file, Set() calls,
// and environment-variable overrides mapped via the configured env prefix
// (so CORE_CONFIG_DEV_EDITOR shows up as "dev.editor").
//
//	for key, value := range cfg.All() {
//	    core.Println(key, value)   // "dev.editor" "vim"
//	}
func (c *Config) All() iter.Seq2[string, any] {
	c.mu.RLock()

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

	values := make(map[string]any, len(keys))
	for _, key := range keys {
		values[key] = c.full.Get(key)
	}
	c.mu.RUnlock()

	slices.Sort(keys)

	return func(yield func(string, any) bool) {
		for _, key := range keys {
			if !yield(key, values[key]) {
				return
			}
		}
	}
}

// snapshotAllLocked copies the current config state while c.mu is already held.
func (c *Config) snapshotAllLocked() map[string]any {
	out := make(map[string]any, len(c.full.AllKeys()))
	for _, key := range c.full.AllKeys() {
		out[key] = c.full.Get(key)
	}
	return out
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
	if !core.HasSuffix(canonical, "_") {
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
	leaf := flattenSettings(nil, source.file.AllSettings())
	for key, value := range flattenSettings(nil, source.full.AllSettings()) {
		if _, ok := leaf[key]; ok {
			continue
		}
		leaf[key] = value
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

func flattenSettings(dst, settings map[string]any) map[string]any {
	if dst == nil {
		dst = map[string]any{}
	}
	flattenInto(dst, "", settings)
	return dst
}

func flattenInto(dst map[string]any, prefix string, value any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, next := range typed {
			flattenInto(dst, joinConfigPath(prefix, key), next)
		}
	case map[any]any:
		for key, next := range typed {
			flattenInto(dst, joinConfigPath(prefix, core.Sprintf("%v", key)), next)
		}
	case []any:
		if prefix != "" {
			dst[prefix] = typed
		}
	case []string:
		if prefix != "" {
			dst[prefix] = typed
		}
	default:
		if prefix != "" {
			dst[prefix] = value
		}
	}
}

func joinConfigPath(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// OnChange registers a callback invoked when a config key changes.
// The callback receives the key path and new value.
// Multiple callbacks can be registered; all are called in order.
//
//	cfg.OnChange(func(key string, value any) {
//	    if key == "dev.editor" {
//	        core.Println("editor changed to", value)
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
func Load(m coreio.Medium, path string) core.Result {
	ext := core.Lower(core.PathExt(path))
	switch ext {
	case "", ".yaml", ".yml":
	// These paths are safe to treat as YAML sources.
	case ".env":
		// dotenv sources are also supported by the RFC contract.
	default:
		if core.PathBase(path) != ".env" {
			return core.Fail(core.E(callerConfigLoad, unsupportedConfigFileType+": "+path, nil))
		}
	}

	content, err := m.Read(path)
	if err != nil {
		return core.Fail(core.E(callerConfigLoad, "failed to read config file: "+path, err))
	}

	v := viper.New()
	switch {
	case ext == ".env" || core.PathBase(path) == ".env":
		v.SetConfigType("env")
	default:
		v.SetConfigType("yaml")
	}
	if err := v.ReadConfig(core.NewReader(content)); err != nil {
		return core.Fail(core.E(callerConfigLoad, "failed to parse config file: "+path, err))
	}

	return core.Ok(v.AllSettings())
}

// Save writes configuration data to a YAML file at the given path.
// It ensures the parent directory exists before writing and uses 0600
// permissions for the file so user config does not become world-readable.
//
//	config.Save(io.Local, "~/.core/config.yaml", map[string]any{"dev": map[string]any{"editor": "vim"}})
func Save(m coreio.Medium, path string, data map[string]any) core.Result {
	switch ext := core.Lower(core.PathExt(path)); ext {
	case "", ".yaml", ".yml":
		// These paths are safe to treat as YAML destinations.
	default:
		return core.Fail(core.E(callerConfigSave, unsupportedConfigFileType+": "+path, nil))
	}

	payload := make(map[string]any, len(data)+1)
	maps.Copy(payload, data)
	// Every .core YAML payload is versioned for forward compatibility.
	payload["version"] = 1

	out, err := yaml.Marshal(payload)
	if err != nil {
		return core.Fail(core.E(callerConfigSave, "failed to marshal config", err))
	}

	dir := core.PathDir(path)
	if err := m.EnsureDir(dir); err != nil {
		return core.Fail(core.E(callerConfigSave, "failed to create config directory: "+dir, err))
	}

	if err := m.WriteMode(path, string(out), 0600); err != nil {
		return core.Fail(core.E(callerConfigSave, "failed to write config file: "+path, err))
	}

	return core.Ok(nil)
}

func persistToStore(store ConfigStoreWriter, key string, value any) {
	if store == nil || key == "" {
		return
	}
	if err := store.Set("config", key, core.JSONMarshalString(value)); err != nil {
		return
	}
}

func (c *Config) loadStoreState() core.Result {
	reader, ok := c.store.(ConfigStoreReader)
	if !ok || reader == nil {
		return core.Ok(nil)
	}

	entries, err := reader.GetAll("config")
	if err != nil {
		return core.Fail(core.E("config.loadStoreState", "failed to read config entries from store", err))
	}

	for key, raw := range entries {
		if key == "" {
			continue
		}
		decoded := decodeStoredConfigValue(raw)
		c.file.Set(key, decoded)
		c.full.Set(key, decoded)
	}

	return core.Ok(nil)
}

func decodeStoredConfigValue(raw string) any {
	if raw == "" {
		return ""
	}

	var decoded any
	if r := core.JSONUnmarshalString(raw, &decoded); r.OK {
		return decoded
	}

	// Older or non-core writers may persist plain strings instead of JSON.
	return raw
}
