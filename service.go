package config

import core "dappco.re/go"

const errConfigNotLoaded = "config not loaded"

// Service wraps Config as a framework service with lifecycle support.
type Service struct {
	*core.ServiceRuntime[ServiceOptions]
	config *Config
}

// ServiceOptions holds configuration for the config service.
type ServiceOptions struct {
	// Path overrides the default config file path.
	Path string
	// EnvPrefix overrides the default environment variable prefix.
	EnvPrefix string
	// Medium overrides the default storage medium.
	Medium Medium
}

// NewConfigService creates a new config service factory for the Core framework.
// Register it with core.WithService(config.NewConfigService).
func NewConfigService(c *core.Core) core.Result {
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{}),
	}
	return core.Ok(svc)
}

// OnStartup loads the configuration file during application startup.
func (s *Service) OnStartup(_ core.Context) core.Result {
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

	r := New(configOpts...)
	if !r.OK {
		return core.Fail(core.E("config.Service.OnStartup", "failed to create config", core.NewError(r.Error())))
	}

	s.config = r.Value.(*Config)
	return core.Ok(nil)
}

// Get retrieves a configuration value by key.
func (s *Service) Get(key string, out any) core.Result {
	if s == nil || s.config == nil {
		return core.Fail(core.E("config.Service.Get", errConfigNotLoaded, nil))
	}
	return s.config.Get(key, out)
}

// Set stores a configuration value by key.
func (s *Service) Set(key string, v any) core.Result {
	if s == nil || s.config == nil {
		return core.Fail(core.E("config.Service.Set", errConfigNotLoaded, nil))
	}
	return s.config.Set(key, v)
}

// Commit persists any configuration changes to disk.
func (s *Service) Commit() core.Result {
	if s == nil || s.config == nil {
		return core.Fail(core.E("config.Service.Commit", errConfigNotLoaded, nil))
	}
	return s.config.Commit()
}

// LoadFile merges a configuration file into the central configuration.
func (s *Service) LoadFile(m Medium, path string) core.Result {
	if s == nil || s.config == nil {
		return core.Fail(core.E("config.Service.LoadFile", errConfigNotLoaded, nil))
	}
	return s.config.LoadFile(m, path)
}

// Ensure Service implements Startable at compile time.
var _ core.Startable = (*Service)(nil)
