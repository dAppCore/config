package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

// featurePrefix is the environment variable prefix for feature flag overrides.
const featurePrefix = "CORE_FEATURE_"

// featureConfigKey is the config-file key under which feature flags live. A
// .core/config.yaml entry `features: { dark-mode: true }` maps to the flag
// `dark-mode`.
const featureConfigKey = "features"

var (
	featureMu      sync.RWMutex
	featureDefault = &featureRegistry{values: map[string]bool{}}
	featureSource  *Config
)

type featureRegistry struct {
	values map[string]bool
}

// Feature returns whether a feature flag is enabled. Checks in order:
// environment variable, loaded config (`features.*`), process-level registry,
// then false.
//
//	if config.Feature("dark-mode") {
//	    theme.UseDark()
//	}
func Feature(name string) bool {
	if v, ok := featureEnv(name); ok {
		return v
	}
	if v, ok := featureFromSource(name); ok {
		return v
	}
	featureMu.RLock()
	defer featureMu.RUnlock()
	return featureDefault.values[name]
}

// FeatureFromConfig checks a specific Config instance for a feature flag without
// touching process-level state. Environment overrides still win.
//
//	if config.FeatureFromConfig(cfg, "beta-api") { mux.Use(betaRoutes) }
func FeatureFromConfig(cfg *Config, name string) bool {
	if v, ok := featureEnv(name); ok {
		return v
	}
	if cfg == nil {
		return false
	}
	return featureLookup(cfg, name)
}

// SetFeatureSource registers a Config instance to back Feature() lookups from
// loaded config files. Pass nil to clear. This lets a Core service publish its
// `features:` tree as the global source without every caller plumbing the cfg.
//
//	cfg, _ := config.Discover()
//	config.SetFeatureSource(cfg)
//	defer config.SetFeatureSource(nil)
func SetFeatureSource(cfg *Config) {
	featureMu.Lock()
	defer featureMu.Unlock()
	featureSource = cfg
}

func featureFromSource(name string) (bool, bool) {
	featureMu.RLock()
	cfg := featureSource
	featureMu.RUnlock()
	if cfg == nil {
		return false, false
	}
	return featureLookup(cfg, name), cfg.hasFeature(name)
}

func featureLookup(cfg *Config, name string) bool {
	cfg.mu.RLock()
	defer cfg.mu.RUnlock()
	key := featureConfigKey + "." + name
	if !cfg.full.IsSet(key) {
		return false
	}
	return cfg.full.GetBool(key)
}

// hasFeature reports whether a flag is explicitly present in the loaded config.
// Used by Feature() to decide whether the config layer should be trusted vs
// falling through to the process-level registry.
func (c *Config) hasFeature(name string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.full.IsSet(featureConfigKey + "." + name)
}

// SetFeature sets a process-level feature flag. Environment variables still win.
//
//	config.SetFeature("dark-mode", true)
func SetFeature(name string, enabled bool) {
	featureMu.Lock()
	defer featureMu.Unlock()
	featureDefault.values[name] = enabled
}

// Features returns the set of enabled feature flag names from the process-level
// registry. Environment overrides are not included in the returned slice.
//
//	for _, flag := range config.Features() {
//	    fmt.Println(flag)
//	}
func Features() []string {
	featureMu.RLock()
	defer featureMu.RUnlock()
	var out []string
	for name, enabled := range featureDefault.values {
		if enabled {
			out = append(out, name)
		}
	}
	return out
}

// featureEnv maps dark-mode → CORE_FEATURE_DARK_MODE.
func featureEnv(name string) (bool, bool) {
	envName := featurePrefix + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	raw, ok := os.LookupEnv(envName)
	if !ok {
		return false, false
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		return false, false
	}
	return b, true
}

// resetFeatureRegistry clears process-level feature state. Test-only helper;
// the exported Feature/SetFeature API is the public contract.
func resetFeatureRegistry() {
	featureMu.Lock()
	defer featureMu.Unlock()
	featureDefault = &featureRegistry{values: map[string]bool{}}
	featureSource = nil
}
