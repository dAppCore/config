package config

import (
	"context"
	"reflect"

	core "dappco.re/go"
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

	optionKeyPath = "pa" + "th"
)

type configOperation func(*Service, *Config, core.Options) core.Result

type serviceLoadPathResolution struct {
	Candidate string
	Core      string
}

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
	return core.Ok(svc)
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
		return core.Ok(svc)
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

	cfgResult := newServiceConfig(opts, configOpts)
	if !cfgResult.OK {
		return core.Fail(coreerr.E("config.Service.OnStartup", "failed to create config", resultCause(cfgResult).(error)))
	}

	s.config = cfgResult.Value.(*Config)

	// Publish the loaded config as the process-wide feature source so
	// config.Feature() reflects the current .core/config.yaml by default.
	SetFeatureSource(s.config)

	if c := s.Core(); c != nil {
		s.config.AttachCore(c)
		s.registerActions(c)
		s.registerCommands(c)
	}

	return core.Ok(nil)
}

func newServiceConfig(opts ServiceOptions, configOpts []Option) core.Result {
	if opts.Path == "" {
		return Discover(configOpts...)
	}
	if isDiscoverableConfigPath(opts.Path) {
		return DiscoverFrom(serviceProjectRoot(opts.Path), configOpts...)
	}
	return New(configOpts...)
}

func isDiscoverableConfigPath(path string) bool {
	clean := core.CleanPath(path, string(core.PathSeparator))
	return core.PathBase(clean) == FileConfig && core.PathBase(core.PathDir(clean)) == Directory
}

// OnShutdown releases any active config-file watcher created during the
// service lifetime so fsnotify descriptors do not leak across restarts.
func (s *Service) OnShutdown(_ context.Context) core.Result {
	if s.config != nil {
		s.config.StopWatch()
	}
	return core.Ok(nil)
}

func resolveValidatedServiceLoadPath(basePath, path string) core.Result {
	cleanResult := validateServiceLoadPathInput(path)
	if !cleanResult.OK {
		return cleanResult
	}
	clean := cleanResult.Value.(string)
	if basePath != "" {
		return resolveServiceLoadPathWithinCore(basePath, clean, path)
	}
	return validateServiceConfigPath(clean)
}

func validateServiceLoadPathInput(path string) core.Result {
	if path == "" {
		return core.Fail(coreerr.E(callerValidateServiceLoadPath, "empty config path", nil))
	}
	if core.PathIsAbs(path) {
		return core.Fail(coreerr.E(callerValidateServiceLoadPath, "absolute config paths are not allowed: "+path, nil))
	}

	clean := core.CleanPath(path, string(core.PathSeparator))
	if clean == "." || clean == ".." || core.HasPrefix(clean, ".."+string(core.PathSeparator)) {
		return core.Fail(coreerr.E(callerValidateServiceLoadPath, "path traversal rejected: "+path, nil))
	}
	if !isProjectCoreRelativePath(clean) {
		return core.Fail(coreerr.E(callerValidateServiceLoadPath, "config paths must remain under .core/: "+path, nil))
	}
	return core.Ok(clean)
}

func resolveServiceLoadPathWithinCore(basePath, clean, original string) core.Result {
	projectRoot := serviceProjectRoot(basePath)
	corePath := core.CleanPath(core.PathJoin(projectRoot, Directory), string(core.PathSeparator))
	absCoreResult := core.PathAbs(corePath)
	if !absCoreResult.OK {
		return core.Fail(coreerr.E(callerValidateServiceLoadPath, "resolve config base failed: "+original, resultCause(absCoreResult).(error)))
	}
	absCorePath := absCoreResult.Value.(string)

	candidatePath := core.CleanPath(core.PathJoin(projectRoot, clean), string(core.PathSeparator))
	absCandidateResult := core.PathAbs(candidatePath)
	if !absCandidateResult.OK {
		return core.Fail(coreerr.E(callerValidateServiceLoadPath, "resolve config path failed: "+original, resultCause(absCandidateResult).(error)))
	}
	absCandidate := absCandidateResult.Value.(string)

	resolutionResult := resolveServiceLoadPath(candidatePath, absCorePath, absCandidate)
	if !resolutionResult.OK {
		return resolutionResult
	}
	resolution := resolutionResult.Value.(serviceLoadPathResolution)
	if r := ensureServicePathInsideCore(resolution.Core, resolution.Candidate, original); !r.OK {
		return r
	}
	return validateServiceConfigPath(candidatePath)
}

func ensureServicePathInsideCore(coreAbs, candidateAbs, original string) core.Result {
	relResult := core.PathRel(coreAbs, candidateAbs)
	if !relResult.OK {
		return core.Fail(coreerr.E(callerValidateServiceLoadPath, "failed relative path check: "+original, resultCause(relResult).(error)))
	}
	rel := relResult.Value.(string)
	if rel != "." && (rel == ".." || core.HasPrefix(rel, ".."+string(core.PathSeparator))) {
		return core.Fail(coreerr.E(callerValidateServiceLoadPath, "config path escapes .core/: "+original, nil))
	}
	return core.Ok(nil)
}

func validateServiceConfigPath(path string) core.Result {
	if r := configTypeForPath(path); !r.OK {
		return r
	}
	return core.Ok(path)
}

func serviceProjectRoot(basePath string) string {
	baseDir := core.CleanPath(core.PathDir(basePath), string(core.PathSeparator))
	if core.PathBase(baseDir) == Directory {
		return core.PathDir(baseDir)
	}
	return baseDir
}

func isProjectCoreRelativePath(path string) bool {
	return path == Directory || core.HasPrefix(path, Directory+string(core.PathSeparator))
}

func resolveServiceLoadPath(candidatePath, coreAbs, absCandidate string) core.Result {
	resolvedCore := coreAbs
	if r := core.Lstat(coreAbs); r.OK && r.Value.(core.FsFileInfo).Mode()&core.ModeSymlink != 0 {
		return core.Fail(coreerr.E(callerValidateServiceLoadPath, "symlinked .core directories are not allowed: "+coreAbs, nil))
	}
	if statResult := core.Stat(coreAbs); statResult.OK && statResult.Value.(core.FsFileInfo).IsDir() {
		if realCoreResult := core.PathEvalSymlinks(coreAbs); realCoreResult.OK {
			resolvedCore = realCoreResult.Value.(string)
		} else {
			return core.Fail(coreerr.E(callerValidateServiceLoadPath, "resolve .core symlink failed: "+coreAbs, resultCause(realCoreResult).(error)))
		}
	}

	resolvedCandidate := absCandidate
	if core.Stat(absCandidate).OK {
		realCandidateResult := core.PathEvalSymlinks(absCandidate)
		if !realCandidateResult.OK {
			return core.Fail(coreerr.E(callerValidateServiceLoadPath, "resolve config symlink failed: "+candidatePath, resultCause(realCandidateResult).(error)))
		}
		resolvedCandidate = realCandidateResult.Value.(string)
	}

	return core.Ok(serviceLoadPathResolution{Candidate: resolvedCandidate, Core: resolvedCore})
}

func resultCause(r core.Result) any {
	if err, ok := r.Value.(error); ok {
		return err
	}
	return core.NewError(r.Error())
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
	_ = c.Command(commandConfigGet, s.configCommand("Read a config value", c, commandConfigGet, configGetOperation))
	_ = c.Command(commandConfigSet, s.configCommand("Set a config value", c, commandConfigSet, configSetOperation))
	_ = c.Command(commandConfigList, s.configCommand("List all config values", c, commandConfigList, configAllOperation))
	_ = c.Command(commandConfigCommit, s.configCommand("Persist config changes", c, commandConfigCommit, configCommitOperation))
	_ = c.Command(commandConfigLoad, s.configCommand("Load a config file", c, commandConfigLoad, configLoadOperation))
	_ = c.Command(commandConfigAll, s.configCommand("List all config values", c, commandConfigAll, configAllOperation))
	_ = c.Command(commandConfigPath, s.configCommand("Show the config file path", c, commandConfigPath, configPathOperation))
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
		return core.Fail(coreerr.E(name, errConfigNotLoaded, nil))
	}
	return op(s, s.config, opts)
}

func configGetOperation(_ *Service, cfg *Config, opts core.Options) core.Result {
	key := opts.String("key")
	var value any
	if r := cfg.Get(key, &value); !r.OK {
		return r
	}
	return core.Ok(value)
}

func configSetOperation(_ *Service, cfg *Config, opts core.Options) core.Result {
	key := opts.String("key")
	r := opts.Get("value")
	if result := cfg.Set(key, r.Value); !result.OK {
		return result
	}
	return core.Ok(nil)
}

func configCommitOperation(_ *Service, cfg *Config, _ core.Options) core.Result {
	if r := cfg.Commit(); !r.OK {
		return r
	}
	return core.Ok(nil)
}

func configLoadOperation(s *Service, cfg *Config, opts core.Options) core.Result {
	if r := s.LoadFile(cfg.medium, opts.String(optionKeyPath)); !r.OK {
		return r
	}
	return core.Ok(nil)
}

func configAllOperation(_ *Service, cfg *Config, _ core.Options) core.Result {
	return core.Ok(configValues(cfg))
}

func configPathOperation(_ *Service, cfg *Config, _ core.Options) core.Result {
	return core.Ok(cfg.Path())
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
func (s *Service) Get(key string, out any) core.Result {
	if s.config == nil {
		return core.Fail(coreerr.E("config.Service.Get", errConfigNotLoaded, nil))
	}
	return s.config.Get(key, out)
}

// Set stores a configuration value by key.
//
//	svc.Set("dev.editor", "vim")
func (s *Service) Set(key string, v any) core.Result {
	if s.config == nil {
		return core.Fail(coreerr.E("config.Service.Set", errConfigNotLoaded, nil))
	}
	return s.config.Set(key, v)
}

// Commit persists any configuration changes to disk.
//
//	svc.Commit()
func (s *Service) Commit() core.Result {
	if s.config == nil {
		return core.Fail(coreerr.E("config.Service.Commit", errConfigNotLoaded, nil))
	}
	return s.config.Commit()
}

// LoadFile merges a configuration file into the central configuration.
//
//	svc.LoadFile(io.Local, ".core/build.yaml")
func (s *Service) LoadFile(m coreio.Medium, path string) core.Result {
	if s.config == nil {
		return core.Fail(coreerr.E("config.Service.LoadFile", errConfigNotLoaded, nil))
	}
	resolvedPathResult := resolveValidatedServiceLoadPath(s.config.Path(), path)
	if !resolvedPathResult.OK {
		return resolvedPathResult
	}
	return s.config.LoadFile(m, resolvedPathResult.Value.(string))
}

func ensureConfigEntitlement(c *core.Core, action string) core.Result {
	if c == nil {
		return core.Fail(coreerr.E(action, "core not available", nil))
	}
	if e := c.Entitled(action); !e.Allowed {
		return core.Fail(coreerr.E(action, "not entitled: "+action+": "+e.Reason, nil))
	}
	return core.Ok(nil)
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
