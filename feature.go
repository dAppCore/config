package config

import (
	"os"
	"strconv"
	"strings"
)

const featureEnvPrefix = "CORE_FEATURE_"

func featureEnvVar(name string) string {
	name = strings.TrimSpace(strings.ToLower(name))
	name = strings.ReplaceAll(name, "-", "_")
	name = strings.ReplaceAll(name, " ", "_")
	return featureEnvPrefix + strings.ToUpper(name)
}

// Feature returns whether a feature flag is enabled.
// Checks environment variables first, then config values, then false.
func Feature(name string) bool {
	envVar := featureEnvVar(name)
	if raw, ok := os.LookupEnv(envVar); ok {
		enabled, err := strconv.ParseBool(raw)
		if err != nil {
			return false
		}
		return enabled
	}

	cfg, err := Discover()
	if err != nil {
		return false
	}

	key := "features." + name
	var enabled bool
	if err := cfg.Get(key, &enabled); err != nil {
		return false
	}
	return enabled
}

// IsFeature is an alias for Feature.
func IsFeature(name string) bool {
	return Feature(name)
}
