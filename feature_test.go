package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFeature_Feature_Good(t *testing.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	assert.False(t, Feature("dark-mode"))
	SetFeature("dark-mode", true)
	assert.True(t, Feature("dark-mode"))
	assert.Contains(t, Features(), "dark-mode")
}

func TestFeature_Feature_Bad(t *testing.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// Unknown flag → false, no panic.
	assert.False(t, Feature("never-declared"))
}

func TestFeature_Feature_Ugly(t *testing.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// Environment override wins over registry, including mapping hyphens to underscores.
	t.Setenv("CORE_FEATURE_DARK_MODE", "true")
	SetFeature("dark-mode", false)
	assert.True(t, Feature("dark-mode"))
}

func TestFeature_SetFeature_Good(t *testing.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	SetFeature("beta-api", true)
	SetFeature("verbose-logging", false)
	flags := Features()
	assert.Contains(t, flags, "beta-api")
	assert.NotContains(t, flags, "verbose-logging")
}
