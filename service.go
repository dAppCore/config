package config

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

const (
	callerValidateServiceLoadPath = "config.validateServiceLoadPath"

	actionConfigGet    = "config.get"
	actionConfigSet    = "config.set"
	actionConfigCommit = "config.commit"
	actionConfigLoad   = "config.load"
	actionConfigAll    = "config.all"
	actionConfigPath   = "config.path"

	commandConfigGet    = "config/get"
	commandConfigSet    = "config/set"
	commandConfigList   = "config/list"
	commandConfigCommit = "config/commit"
	commandConfigLoad   = "config/load"
	commandConfigAll    = "config/all"
	commandConfigPath   = "config/path"

	errConfigNotLoaded = "config not loaded"
)

type configOperation func(*Service, *Config, core.Options) core.Result

// Service wraps Config as a framework service with lifecycle support.
//
//	c := core.New(core.WithService(config.NewConfigService))
//	svc := c.Config()
type Service struct {
	*core.ServiceRuntime[ServiceOptions]
	config *Config
	store  ConfigStoreWriter
}

// ServiceOptions holds configuration for the config service.
type ServiceOptions struct {
	// Path overrides the default config file path.
	Path string
	// EnvPrefix overrides the default environment variable prefix.
	EnvPrefix string
	// Medium overrides the default storage medium.
	Medium coreio.Medium
	// Store mirrors Set() calls into go-store when present.
	Store ConfigStoreWriter
}

// NewConfigService creates a new config service factory for the Core framework.
// Register it with core.WithService(config.NewConfigService). The returned
// Result carries the *Service instance so core.WithService can auto-discover
// the "config" service name from the package path and wire it into the
// lifecycle and IPC bus.
//
//	c := core.New(core.WithService(config.NewConfigService))
//	svc, _ := core.ServiceFor[*config.Service](c, "config")
func NewConfigService(c *core.Core) core.Result {
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{}),
	}
	return core.Result{Value: svc, OK: true}
}

// NewConfigServiceWith returns a service factory pre-populated with the given
// options. Use this when the default path / medium / env prefix aren't right
// for the host application.
//
//	c := core.New(core.WithService(config.NewConfigServiceWith(config.ServiceOptions{
//	    Path: "/etc/myapp/config.yaml",
//	    EnvPrefix: "MYAPP",
//	})))
func NewConfigServiceWith(opts ServiceOptions) func(*core.Core) core.Result {
	return func(c *core.Core) core.Result {
		svc := &Service{
			ServiceRuntime: core.NewServiceRuntime(c, opts),
		}
		return core.Result{Value: svc, OK: true}
	}
}

// OnStartup loads the configuration file during application startup
// and registers named actions and commands on the Core.
//
//	func (s *Service) OnStartup(ctx context.Context) core.Result
func (s *Service) OnStartup(_ context.Context) core.Result {
	opts := s.Options()

	var configOpts []Option
	if opts.Path != "" {
		configOpts = append(configOpts, WithPath(opts.Path))
	}
	if opts.EnvPrefix != "" {
		configOpts = append(configOpts, WithEnvPrefix(opts.EnvPrefix))
	}
	if opts.Medium != nil {
		configOpts = append(configOpts, WithMedium(opts.Medium))
	}
	s.store = opts.Store
	if s.store == nil {
		s.store = discoverStoreWriter(s.Core())
	}
	if s.store != nil {
		configOpts = append(configOpts, WithStore(s.store))
	}

	cfg, err := newServiceConfig(opts, configOpts)
	if err != nil {
		return core.Result{Value: coreerr.E("config.Service.OnStartup", "failed to create config", err), OK: false}
	}

	s.config = cfg

	// Publish the loaded config as the process-wide feature source so
	// config.Feature() reflects the current .core/config.yaml by default.
	SetFeatureSource(cfg)

	if c := s.Core(); c != nil {
		s.config.AttachCore(c)
		s.registerActions(c)
		s.registerCommands(c)
	}

	return core.Result{OK: true}
}

func newServiceConfig(opts ServiceOptions, configOpts []Option) (*Config, error) {
	if opts.Path == "" {
		return Discover(configOpts...)
	}
	if isDiscoverableConfigPath(opts.Path) {
		return DiscoverFrom(serviceProjectRoot(opts.Path), configOpts...)
	}
	return New(configOpts...)
}

func isDiscoverableConfigPath(path string) bool {
	clean := filepath.Clean(path)
	return filepath.Base(clean) == FileConfig && filepath.Base(filepath.Dir(clean)) == Directory
}

// OnShutdown releases any active config-file watcher created during the
// service lifetime so fsnotify descriptors do not leak across restarts.
func (s *Service) OnShutdown(_ context.Context) core.Result {
	if s.config != nil {
		s.config.StopWatch()
	}
	return core.Result{OK: true}
}

func resolveValidatedServiceLoadPath(basePath, path string) (string, error) {
	clean, err := validateServiceLoadPathInput(path)
	if err != nil {
		return "", err
	}
	if basePath != "" {
		return resolveServiceLoadPathWithinCore(basePath, clean, path)
	}
	return validateServiceConfigPath(clean)
}

func validateServiceLoadPathInput(path string) (string, error) {
	if path == "" {
		return "", coreerr.E(callerValidateServiceLoadPath, "empty config path", nil)
	}
	if filepath.IsAbs(path) {
		return "", coreerr.E(callerValidateServiceLoadPath, "absolute config paths are not allowed: "+path, nil)
	}

	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", coreerr.E(callerValidateServiceLoadPath, "path traversal rejected: "+path, nil)
	}
	if !isProjectCoreRelativePath(clean) {
		return "", coreerr.E(callerValidateServiceLoadPath, "config paths must remain under .core/: "+path, nil)
	}
	return clean, nil
}

func resolveServiceLoadPathWithinCore(basePath, clean, original string) (string, error) {
	projectRoot := serviceProjectRoot(basePath)
	corePath := filepath.Clean(filepath.Join(projectRoot, Directory))
	absCorePath, err := filepath.Abs(corePath)
	if err != nil {
		return "", coreerr.E(callerValidateServiceLoadPath, "resolve config base failed: "+original, err)
	}

	candidatePath := filepath.Clean(filepath.Join(projectRoot, clean))
	absCandidate, err := filepath.Abs(candidatePath)
	if err != nil {
		return "", coreerr.E(callerValidateServiceLoadPath, "resolve config path failed: "+original, err)
	}

	resolvedCandidate, resolvedCore, err := resolveServiceLoadPath(candidatePath, absCorePath, absCandidate)
	if err != nil {
		return "", err
	}
	if err := ensureServicePathInsideCore(resolvedCore, resolvedCandidate, original); err != nil {
		return "", err
	}
	return validateServiceConfigPath(candidatePath)
}

func ensureServicePathInsideCore(coreAbs, candidateAbs, original string) error {
	rel, err := filepath.Rel(coreAbs, candidateAbs)
	if err != nil {
		return coreerr.E(callerValidateServiceLoadPath, "failed relative path check: "+original, err)
	}
	if rel != "." && (rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
		return coreerr.E(callerValidateServiceLoadPath, "config path escapes .core/: "+original, nil)
	}
	return nil
}

func validateServiceConfigPath(path string) (string, error) {
	if _, err := configTypeForPath(path); err != nil {
		return "", err
	}
	return path, nil
}

func serviceProjectRoot(basePath string) string {
	baseDir := filepath.Clean(filepath.Dir(basePath))
	if filepath.Base(baseDir) == Directory {
		return filepath.Dir(baseDir)
	}
	return baseDir
}

func isProjectCoreRelativePath(path string) bool {
	return path == Directory || strings.HasPrefix(path, Directory+string(filepath.Separator))
}

func resolveServiceLoadPath(candidatePath, coreAbs, absCandidate string) (string, string, error) {
	resolvedCore := coreAbs
	if info, err := os.Lstat(coreAbs); err == nil && info.Mode()&os.ModeSymlink != 0 {
		return "", "", coreerr.E(callerValidateServiceLoadPath, "symlinked .core directories are not allowed: "+coreAbs, nil)
	}
	if stat, err := os.Stat(coreAbs); err == nil && stat.IsDir() {
		if realCore, err := filepath.EvalSymlinks(coreAbs); err == nil {
			resolvedCore = realCore
		} else {
			return "", "", coreerr.E(callerValidateServiceLoadPath, "resolve .core symlink failed: "+coreAbs, err)
		}
	}

	resolvedCandidate := absCandidate
	if _, err := os.Stat(absCandidate); err == nil {
		realCandidate, err := filepath.EvalSymlinks(absCandidate)
		if err != nil {
			return "", "", coreerr.E(callerValidateServiceLoadPath, "resolve config symlink failed: "+candidatePath, err)
		}
		resolvedCandidate = realCandidate
	}

	return resolvedCandidate, resolvedCore, nil
}

// registerActions exposes config.get/set/commit/load/all on the Core IPC bus.
//
//	c.Action("config.get").Run(ctx, core.NewOptions(core.Option{Key:"key", Value:"dev.editor"}))
func (s *Service) registerActions(c *core.Core) {
	c.Action(actionConfigGet, s.actionHandler(c, actionConfigGet, configGetOperation))
	c.Action(actionConfigSet, s.actionHandler(c, actionConfigSet, configSetOperation))
	c.Action(actionConfigCommit, s.actionHandler(c, actionConfigCommit, configCommitOperation))
	c.Action(actionConfigLoad, s.actionHandler(c, actionConfigLoad, configLoadOperation))
	c.Action(actionConfigAll, s.actionHandler(c, actionConfigAll, configAllOperation))
	c.Action(actionConfigPath, s.actionHandler(c, actionConfigPath, configPathOperation))
}

// registerCommands exposes config commands for CLI discovery.
//
//	core config/get --key dev.editor
func (s *Service) registerCommands(c *core.Core) {
	c.Command(commandConfigGet, s.configCommand("Read a config value", c, commandConfigGet, configGetOperation))
	c.Command(commandConfigSet, s.configCommand("Set a config value", c, commandConfigSet, configSetOperation))
	c.Command(commandConfigList, s.configCommand("List all config values", c, commandConfigList, configAllOperation))
	c.Command(commandConfigCommit, s.configCommand("Persist config changes", c, commandConfigCommit, configCommitOperation))
	c.Command(commandConfigLoad, s.configCommand("Load a config file", c, commandConfigLoad, configLoadOperation))
	c.Command(commandConfigAll, s.configCommand("List all config values", c, commandConfigAll, configAllOperation))
	c.Command(commandConfigPath, s.configCommand("Show the config file path", c, commandConfigPath, configPathOperation))
}

func (s *Service) actionHandler(c *core.Core, name string, op configOperation) core.ActionHandler {
	return func(_ context.Context, opts core.Options) core.Result {
		return s.runConfigOperation(c, name, opts, op)
	}
}

func (s *Service) configCommand(description string, c *core.Core, name string, op configOperation) core.Command {
	return core.Command{
		Description: description,
		Action: func(opts core.Options) core.Result {
			return s.runConfigOperation(c, name, opts, op)
		},
	}
}

func (s *Service) runConfigOperation(c *core.Core, name string, opts core.Options, op configOperation) core.Result {
	if result := ensureConfigEntitlement(c, name); !result.OK {
		return result
	}
	if s.config == nil {
		return core.Result{Value: coreerr.E(name, errConfigNotLoaded, nil), OK: false}
	}
	return op(s, s.config, opts)
}

func configGetOperation(_ *Service, cfg *Config, opts core.Options) core.Result {
	key := opts.String("key")
	var value any
	if err := cfg.Get(key, &value); err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{Value: value, OK: true}
}

func configSetOperation(_ *Service, cfg *Config, opts core.Options) core.Result {
	key := opts.String("key")
	r := opts.Get("value")
	if err := cfg.Set(key, r.Value); err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
}

func configCommitOperation(_ *Service, cfg *Config, _ core.Options) core.Result {
	if err := cfg.Commit(); err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
}

func configLoadOperation(s *Service, cfg *Config, opts core.Options) core.Result {
	if err := s.LoadFile(cfg.medium, opts.String("path")); err != nil {
		return core.Result{Value: err, OK: false}
	}
	return core.Result{OK: true}
}

func configAllOperation(_ *Service, cfg *Config, _ core.Options) core.Result {
	return core.Result{Value: configValues(cfg), OK: true}
}

func configPathOperation(_ *Service, cfg *Config, _ core.Options) core.Result {
	return core.Result{Value: cfg.Path(), OK: true}
}

func configValues(cfg *Config) map[string]any {
	out := make(map[string]any)
	for k, v := range cfg.All() {
		out[k] = v
	}
	return out
}

// Get retrieves a configuration value by key.
//
//	var editor string
//	svc.Get("dev.editor", &editor)
func (s *Service) Get(key string, out any) error {
	if s.config == nil {
		return coreerr.E("config.Service.Get", errConfigNotLoaded, nil)
	}
	return s.config.Get(key, out)
}

// Set stores a configuration value by key.
//
//	svc.Set("dev.editor", "vim")
func (s *Service) Set(key string, v any) error {
	if s.config == nil {
		return coreerr.E("config.Service.Set", errConfigNotLoaded, nil)
	}
	return s.config.Set(key, v)
}

// Commit persists any configuration changes to disk.
//
//	svc.Commit()
func (s *Service) Commit() error {
	if s.config == nil {
		return coreerr.E("config.Service.Commit", errConfigNotLoaded, nil)
	}
	return s.config.Commit()
}

// LoadFile merges a configuration file into the central configuration.
//
//	svc.LoadFile(io.Local, ".core/build.yaml")
func (s *Service) LoadFile(m coreio.Medium, path string) error {
	if s.config == nil {
		return coreerr.E("config.Service.LoadFile", errConfigNotLoaded, nil)
	}
	resolvedPath, err := resolveValidatedServiceLoadPath(s.config.Path(), path)
	if err != nil {
		return err
	}
	return s.config.LoadFile(m, resolvedPath)
}

func ensureConfigEntitlement(c *core.Core, action string) core.Result {
	if c == nil {
		return core.Result{Value: coreerr.E(action, "core not available", nil), OK: false}
	}
	if e := c.Entitled(action); !e.Allowed {
		return core.Result{Value: coreerr.E(action, "not entitled: "+action+": "+e.Reason, nil), OK: false}
	}
	return core.Result{OK: true}
}

// Config returns the underlying Config instance for advanced operations.
//
//	cfg := svc.Config()
//	cfg.OnChange(func(k string, v any) { ... })
func (s *Service) Config() *Config {
	return s.config
}

func discoverStoreWriter(c *core.Core) ConfigStoreWriter {
	if c == nil {
		return nil
	}

	result := c.Service("store")
	if !result.OK || result.Value == nil {
		return nil
	}
	if store, ok := result.Value.(ConfigStoreWriter); ok {
		return store
	}

	ref := reflect.ValueOf(result.Value)
	if !ref.IsValid() {
		return nil
	}
	if ref.Kind() == reflect.Ptr && !ref.IsNil() {
		ref = ref.Elem()
	}
	if ref.Kind() != reflect.Struct {
		return nil
	}

	field := ref.FieldByName("Store")
	if !field.IsValid() || !field.CanInterface() {
		return nil
	}
	if (field.Kind() == reflect.Ptr || field.Kind() == reflect.Interface) && field.IsNil() {
		return nil
	}
	store, ok := field.Interface().(ConfigStoreWriter)
	if !ok {
		return nil
	}
	return store
}

// Ensure Service implements lifecycle contracts at compile time.
var _ core.Startable = (*Service)(nil)
var _ core.Stoppable = (*Service)(nil)
