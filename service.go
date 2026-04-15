package config

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

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

	cfg, err := New(configOpts...)
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

func validateServiceLoadPath(basePath, path string) error {
	if path == "" {
		return coreerr.E("config.validateServiceLoadPath", "empty config path", nil)
	}
	if filepath.IsAbs(path) {
		return coreerr.E("config.validateServiceLoadPath", "absolute config paths are not allowed: "+path, nil)
	}

	clean := filepath.Clean(path)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return coreerr.E("config.validateServiceLoadPath", "path traversal rejected: "+path, nil)
	}
	if !strings.Contains(clean, string(filepath.Separator)+Directory+string(filepath.Separator)) &&
		!strings.HasPrefix(clean, Directory+string(filepath.Separator)) &&
		clean != Directory {
		return coreerr.E("config.validateServiceLoadPath", "config paths must remain under .core/: "+path, nil)
	}
	validatedPathType := false
	if basePath != "" {
		base := filepath.Clean(filepath.Dir(basePath))
		corePath := filepath.Clean(filepath.Join(base, Directory))
		absCorePath, err := filepath.Abs(corePath)
		if err != nil {
			return coreerr.E("config.validateServiceLoadPath", "resolve config base failed: "+path, err)
		}
		candidatePath := filepath.Clean(filepath.Join(base, clean))
		absCandidate, err := filepath.Abs(candidatePath)
		if err != nil {
			return coreerr.E("config.validateServiceLoadPath", "resolve config path failed: "+path, err)
		}

		resolvedCandidate, resolvedCore, err := resolveServiceLoadPath(candidatePath, absCorePath, absCandidate)
		if err != nil {
			return err
		}
		coreAbs := resolvedCore
		rel, relErr := filepath.Rel(coreAbs, resolvedCandidate)
		if relErr != nil {
			return coreerr.E("config.validateServiceLoadPath", "failed relative path check: "+path, relErr)
		}
		if rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))) {
			validatedPathType = true
		} else {
			return coreerr.E("config.validateServiceLoadPath", "config path escapes .core/: "+path, nil)
		}
	}
	if basePath == "" {
		validatedPathType = true
	}
	if !validatedPathType {
		return coreerr.E("config.validateServiceLoadPath", "config path validation failed: "+path, nil)
	}
	if _, err := configTypeForPath(clean); err != nil {
		return err
	}
	return nil
}

func resolveServiceLoadPath(candidatePath, coreAbs, absCandidate string) (string, string, error) {
	resolvedCore := coreAbs
	if stat, err := os.Stat(coreAbs); err == nil && stat.IsDir() {
		if realCore, err := filepath.EvalSymlinks(coreAbs); err == nil {
			resolvedCore = realCore
		} else {
			return "", "", coreerr.E("config.validateServiceLoadPath", "resolve .core symlink failed: "+coreAbs, err)
		}
	}

	resolvedCandidate := absCandidate
	if _, err := os.Stat(absCandidate); err == nil {
		realCandidate, err := filepath.EvalSymlinks(absCandidate)
		if err != nil {
			return "", "", coreerr.E("config.validateServiceLoadPath", "resolve config symlink failed: "+candidatePath, err)
		}
		resolvedCandidate = realCandidate
	}

	return resolvedCandidate, resolvedCore, nil
}

// registerActions exposes config.get/set/commit/load/all on the Core IPC bus.
//
//	c.Action("config.get").Run(ctx, core.NewOptions(core.Option{Key:"key", Value:"dev.editor"}))
func (s *Service) registerActions(c *core.Core) {
	c.Action("config.get", func(_ context.Context, opts core.Options) core.Result {
		key := opts.String("key")
		if s.config == nil {
			return core.Result{Value: coreerr.E("config.get", "config not loaded", nil), OK: false}
		}
		var value any
		if err := s.config.Get(key, &value); err != nil {
			return core.Result{Value: err, OK: false}
		}
		return core.Result{Value: value, OK: true}
	})

	c.Action("config.set", func(_ context.Context, opts core.Options) core.Result {
		if err := ensureConfigEntitlement(c, "config.set"); err != nil {
			return err
		}
		key := opts.String("key")
		r := opts.Get("value")
		if s.config == nil {
			return core.Result{Value: coreerr.E("config.set", "config not loaded", nil), OK: false}
		}
		if err := s.config.Set(key, r.Value); err != nil {
			return core.Result{Value: err, OK: false}
		}
		return core.Result{OK: true}
	})

	c.Action("config.commit", func(_ context.Context, _ core.Options) core.Result {
		if err := ensureConfigEntitlement(c, "config.commit"); err != nil {
			return err
		}
		if s.config == nil {
			return core.Result{Value: coreerr.E("config.commit", "config not loaded", nil), OK: false}
		}
		if err := s.config.Commit(); err != nil {
			return core.Result{Value: err, OK: false}
		}
		return core.Result{OK: true}
	})

	c.Action("config.load", func(_ context.Context, opts core.Options) core.Result {
		if err := ensureConfigEntitlement(c, "config.load"); err != nil {
			return err
		}
		path := opts.String("path")
		if s.config == nil {
			return core.Result{Value: coreerr.E("config.load", "config not loaded", nil), OK: false}
		}
		if err := s.LoadFile(s.config.medium, path); err != nil {
			return core.Result{Value: err, OK: false}
		}
		return core.Result{OK: true}
	})

	c.Action("config.all", func(_ context.Context, _ core.Options) core.Result {
		if err := ensureConfigEntitlement(c, "config.all"); err != nil {
			return err
		}
		if s.config == nil {
			return core.Result{Value: coreerr.E("config.all", "config not loaded", nil), OK: false}
		}
		out := make(map[string]any)
		for k, v := range s.config.All() {
			out[k] = v
		}
		return core.Result{Value: out, OK: true}
	})

	c.Action("config.path", func(_ context.Context, _ core.Options) core.Result {
		if err := ensureConfigEntitlement(c, "config.path"); err != nil {
			return err
		}
		if s.config == nil {
			return core.Result{Value: coreerr.E("config.path", "config not loaded", nil), OK: false}
		}
		return core.Result{Value: s.config.Path(), OK: true}
	})
}

// registerCommands exposes config commands for CLI discovery.
//
//	core config/get --key dev.editor
func (s *Service) registerCommands(c *core.Core) {
	c.Command("config/get", core.Command{
		Description: "Read a config value",
		Action: func(opts core.Options) core.Result {
			key := opts.String("key")
			if s.config == nil {
				return core.Result{Value: coreerr.E("config/get", "config not loaded", nil), OK: false}
			}
			var value any
			if err := s.config.Get(key, &value); err != nil {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{Value: value, OK: true}
		},
	})

	c.Command("config/set", core.Command{
		Description: "Set a config value",
		Action: func(opts core.Options) core.Result {
			if err := ensureConfigEntitlement(c, "config/set"); err != nil {
				return err
			}
			key := opts.String("key")
			r := opts.Get("value")
			if s.config == nil {
				return core.Result{Value: coreerr.E("config/set", "config not loaded", nil), OK: false}
			}
			if err := s.config.Set(key, r.Value); err != nil {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{OK: true}
		},
	})

	c.Command("config/list", core.Command{
		Description: "List all config values",
		Action: func(_ core.Options) core.Result {
			if s.config == nil {
				return core.Result{Value: coreerr.E("config/list", "config not loaded", nil), OK: false}
			}
			out := make(map[string]any)
			for k, v := range s.config.All() {
				out[k] = v
			}
			return core.Result{Value: out, OK: true}
		},
	})

	c.Command("config/commit", core.Command{
		Description: "Persist config changes",
		Action: func(_ core.Options) core.Result {
			if err := ensureConfigEntitlement(c, "config/commit"); err != nil {
				return err
			}
			if s.config == nil {
				return core.Result{Value: coreerr.E("config/commit", "config not loaded", nil), OK: false}
			}
			if err := s.config.Commit(); err != nil {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{OK: true}
		},
	})

	c.Command("config/load", core.Command{
		Description: "Load a config file",
		Action: func(opts core.Options) core.Result {
			if err := ensureConfigEntitlement(c, "config/load"); err != nil {
				return err
			}
			if s.config == nil {
				return core.Result{Value: coreerr.E("config/load", "config not loaded", nil), OK: false}
			}
			path := opts.String("path")
			if err := s.LoadFile(s.config.medium, path); err != nil {
				return core.Result{Value: err, OK: false}
			}
			return core.Result{OK: true}
		},
	})

	c.Command("config/all", core.Command{
		Description: "List all config values",
		Action: func(_ core.Options) core.Result {
			if err := ensureConfigEntitlement(c, "config/all"); err != nil {
				return err
			}
			if s.config == nil {
				return core.Result{Value: coreerr.E("config/all", "config not loaded", nil), OK: false}
			}
			out := make(map[string]any)
			for k, v := range s.config.All() {
				out[k] = v
			}
			return core.Result{Value: out, OK: true}
		},
	})

	c.Command("config/path", core.Command{
		Description: "Show the config file path",
		Action: func(_ core.Options) core.Result {
			if err := ensureConfigEntitlement(c, "config/path"); err != nil {
				return err
			}
			if s.config == nil {
				return core.Result{Value: coreerr.E("config/path", "config not loaded", nil), OK: false}
			}
			return core.Result{Value: s.config.Path(), OK: true}
		},
	})
}

// Get retrieves a configuration value by key.
//
//	var editor string
//	svc.Get("dev.editor", &editor)
func (s *Service) Get(key string, out any) error {
	if s.config == nil {
		return coreerr.E("config.Service.Get", "config not loaded", nil)
	}
	return s.config.Get(key, out)
}

// Set stores a configuration value by key.
//
//	svc.Set("dev.editor", "vim")
func (s *Service) Set(key string, v any) error {
	if s.config == nil {
		return coreerr.E("config.Service.Set", "config not loaded", nil)
	}
	return s.config.Set(key, v)
}

// Commit persists any configuration changes to disk.
//
//	svc.Commit()
func (s *Service) Commit() error {
	if s.config == nil {
		return coreerr.E("config.Service.Commit", "config not loaded", nil)
	}
	return s.config.Commit()
}

// LoadFile merges a configuration file into the central configuration.
//
//	svc.LoadFile(io.Local, ".core/build.yaml")
func (s *Service) LoadFile(m coreio.Medium, path string) error {
	if s.config == nil {
		return coreerr.E("config.Service.LoadFile", "config not loaded", nil)
	}
	if err := validateServiceLoadPath(s.config.Path(), path); err != nil {
		return err
	}
	return s.config.LoadFile(m, path)
}

func ensureConfigEntitlement(c *core.Core, action string) core.Result {
	if c == nil {
		return core.Result{Value: coreerr.E("config."+action, "core not available", nil), OK: false}
	}
	if e := c.Entitled(action); !e.Allowed {
		return core.Result{Value: coreerr.E("config."+action, "not entitled: "+action+": "+e.Reason, nil), OK: false}
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
	if field.Kind() != reflect.Ptr && field.Kind() != reflect.Interface {
		return nil
	}
	if field.IsNil() {
		return nil
	}
	store, _ := field.Interface().(ConfigStoreWriter)
	return store
}

// Ensure Service implements Startable at compile time.
var _ core.Startable = (*Service)(nil)
