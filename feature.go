package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

// featurePrefix is the environment variable prefix for feature flag overrides.
const featurePrefix = "CORE_FEATURE_"

var (
	featureMu      sync.RWMutex
	featureDefault = &featureRegistry{values: map[string]bool{}}
)

type featureRegistry struct {
	values map[string]bool
}

// Feature returns whether a feature flag is enabled. Checks in order:
// environment variable, process-level feature registry, then false.
//
//	if config.Feature("dark-mode") {
//	    theme.UseDark()
//	}
func Feature(name string) bool {
	if v, ok := featureEnv(name); ok {
		return v
	}
	featureMu.RLock()
	defer featureMu.RUnlock()
	return featureDefault.values[name]
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
}
