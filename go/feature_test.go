package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func TestFeature_Feature_Good(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	core.AssertFalse(t, Feature("dark-mode"))
	SetFeature("dark-mode", true)
	core.AssertTrue(t, Feature("dark-mode"))
	core.AssertContains(t, Features(), "dark-mode")
}

func TestFeature_Feature_Bad(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// Unknown flag → false, no panic.
	core.AssertFalse(t, Feature("never-declared"))
}

func TestFeature_Feature_Ugly(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// Environment override wins over registry, including mapping hyphens to underscores.
	t.Setenv("CORE_FEATURE_DARK_MODE", "true")
	SetFeature("dark-mode", false)
	core.AssertTrue(t, Feature("dark-mode"))
}

func TestFeature_SetFeature_Good(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	SetFeature("beta-api", true)
	SetFeature("verbose-logging", false)
	flags := Features()
	core.AssertContains(t, flags, "beta-api")
	core.AssertNotContains(t, flags, "verbose-logging")
}

func TestFeature_FeatureFromConfig_LoadsConfig_Good(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// A loaded config with `features.dark-mode: true` enables the flag without
	// any env var or process-level SetFeature call.
	m := coreio.NewMockMedium()
	m.Files["/cfg.yaml"] = "features:\n  dark-mode: true\n  beta-api: false\n"

	cfg, err := configResult(New(WithMedium(m), WithPath("/cfg.yaml")))
	core.AssertNoError(t, err)

	core.AssertTrue(t, FeatureFromConfig(cfg, "dark-mode"))
	core.AssertFalse(t, FeatureFromConfig(cfg, "beta-api"))
	core.AssertFalse(t, FeatureFromConfig(cfg, "never-declared"))
}

func TestFeature_FeatureFromConfig_NilConfig_Bad(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// Nil config must never panic; returns false for every flag.
	core.AssertFalse(t, FeatureFromConfig(nil, "dark-mode"))
}

func TestFeature_FeatureFromConfig_EnvOverride_Ugly(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// Environment override still wins over a loaded config value.
	t.Setenv("CORE_FEATURE_DARK_MODE", "false")

	m := coreio.NewMockMedium()
	m.Files["/cfg.yaml"] = "features:\n  dark-mode: true\n"
	cfg, err := configResult(New(WithMedium(m), WithPath("/cfg.yaml")))
	core.AssertNoError(t, err)

	core.AssertFalse(t, FeatureFromConfig(cfg, "dark-mode"))
}

func TestFeature_SetFeatureSource_Good(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	m := coreio.NewMockMedium()
	m.Files["/cfg.yaml"] = "features:\n  dark-mode: true\n"
	cfg, err := configResult(New(WithMedium(m), WithPath("/cfg.yaml")))
	core.AssertNoError(t, err)

	// Before registering the source, the flag is false (default registry).
	core.AssertFalse(t, Feature("dark-mode"))

	SetFeatureSource(cfg)
	t.Cleanup(func() { SetFeatureSource(nil) })
	core.AssertTrue(t, Feature("dark-mode"))
}

func TestFeature_SetFeatureSource_Bad(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)

	// Registering a nil source is a safe reset — no panic on lookup afterwards.
	SetFeatureSource(nil)
	core.AssertFalse(t, Feature("dark-mode"))
}

func TestFeature_FeatureFromConfig_Good(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)
	m := coreio.NewMockMedium()
	core.RequireNoError(t, m.Write("/ax7/features.yaml", "features:\n  dark-mode: true\n"))
	cfg, err := configResult(New(WithMedium(m), WithPath("/ax7/features.yaml")))
	core.RequireNoError(t, err)

	got := FeatureFromConfig(cfg, "dark-mode")
	core.AssertTrue(t, got)
}

func TestFeature_FeatureFromConfig_Bad(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)
	got := FeatureFromConfig(nil, "dark-mode")
	core.AssertFalse(t, got)
}

func TestFeature_FeatureFromConfig_Ugly(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)
	t.Setenv("CORE_FEATURE_DARK_MODE", "true")
	got := FeatureFromConfig(nil, "dark-mode")
	core.AssertTrue(t, got)
}

func TestFeature_SetFeatureSource_Ugly(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)
	first, err := configResult(New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/first.yaml"), WithDefaults(map[string]any{"features.dark-mode": true})))
	core.RequireNoError(t, err)
	second, err := configResult(New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/second.yaml"), WithDefaults(map[string]any{"features.dark-mode": false})))
	core.RequireNoError(t, err)

	SetFeatureSource(first)
	SetFeatureSource(second)
	core.AssertFalse(t, Feature("dark-mode"))
}

func TestFeature_SetFeature_Bad(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)
	SetFeature("dark-mode", false)
	got := Feature("dark-mode")
	core.AssertFalse(t, got)
}

func TestFeature_SetFeature_Ugly(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)
	SetFeature("", true)
	got := Features()
	core.AssertContains(t, got, "")
}

func TestFeature_Features_Good(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)
	SetFeature("alpha", true)
	SetFeature("beta", true)
	got := Features()
	core.AssertContains(t, got, "alpha")
}

func TestFeature_Features_Bad(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)
	SetFeature("alpha", false)
	got := Features()
	core.AssertNotContains(t, got, "alpha")
}

func TestFeature_Features_Ugly(t *core.T) {
	resetFeatureRegistry()
	t.Cleanup(resetFeatureRegistry)
	got := Features()
	core.AssertEmpty(t, got)
}
