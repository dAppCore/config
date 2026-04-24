package config

import (
	"testing"

	coreio "dappco.re/go/io"
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

func TestFeature_FromConfig_Good(t *testing.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// A loaded config with `features.dark-mode: true` enables the flag without
	// any env var or process-level SetFeature call.
	m := coreio.NewMockMedium()
	m.Files["/cfg.yaml"] = "features:\n  dark-mode: true\n  beta-api: false\n"

	cfg, err := New(WithMedium(m), WithPath("/cfg.yaml"))
	assert.NoError(t, err)

	assert.True(t, FeatureFromConfig(cfg, "dark-mode"))
	assert.False(t, FeatureFromConfig(cfg, "beta-api"))
	assert.False(t, FeatureFromConfig(cfg, "never-declared"))
}

func TestFeature_FromConfig_Bad(t *testing.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// Nil config must never panic; returns false for every flag.
	assert.False(t, FeatureFromConfig(nil, "dark-mode"))
}

func TestFeature_FromConfig_Ugly(t *testing.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// Environment override still wins over a loaded config value.
	t.Setenv("CORE_FEATURE_DARK_MODE", "false")

	m := coreio.NewMockMedium()
	m.Files["/cfg.yaml"] = "features:\n  dark-mode: true\n"
	cfg, err := New(WithMedium(m), WithPath("/cfg.yaml"))
	assert.NoError(t, err)

	assert.False(t, FeatureFromConfig(cfg, "dark-mode"))
}

func TestFeature_SetFeatureSource_Good(t *testing.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	m := coreio.NewMockMedium()
	m.Files["/cfg.yaml"] = "features:\n  dark-mode: true\n"
	cfg, err := New(WithMedium(m), WithPath("/cfg.yaml"))
	assert.NoError(t, err)

	// Before registering the source, the flag is false (default registry).
	assert.False(t, Feature("dark-mode"))

	SetFeatureSource(cfg)
	t.Cleanup(func() { SetFeatureSource(nil) })
	assert.True(t, Feature("dark-mode"))
}

func TestFeature_SetFeatureSource_Bad(t *testing.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// Registering a nil source is a safe reset — no panic on lookup afterwards.
	SetFeatureSource(nil)
	assert.False(t, Feature("dark-mode"))
}
