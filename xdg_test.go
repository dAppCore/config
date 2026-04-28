package config

import (
	core "dappco.re/go"
	"path/filepath"
	"strings"
)

func TestXdg_XDG_Good(t *core.T) {
	paths := XDG()
	core.AssertEqual(t, "core", paths.Prefix())
	core.AssertTrue(t, strings.HasSuffix(paths.Config(), "/core") || strings.HasSuffix(paths.Config(), "\\core"))
	core.AssertTrue(t, strings.HasSuffix(paths.Data(), "/core") || strings.HasSuffix(paths.Data(), "\\core"))
	core.AssertTrue(t, strings.HasSuffix(paths.Cache(), "/core") || strings.HasSuffix(paths.Cache(), "\\core"))
	core.AssertTrue(t, strings.HasSuffix(paths.Runtime(), "/core") || strings.HasSuffix(paths.Runtime(), "\\core"))
}

func TestXdg_XDG_Bad(t *core.T) {
	// An empty prefix falls back to the default "core" — no panic, no empty paths.
	paths := XDGWithPrefix("")
	configPath := paths.Config()
	core.AssertEqual(t, "core", paths.Prefix())
	core.AssertContains(t, configPath, "core")
}

func TestXdg_XDG_Ugly(t *core.T) {
	// Overriding XDG_CONFIG_HOME via env must change the resolved Config dir.
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	paths := XDGWithPrefix("myapp")
	core.AssertTrue(t, strings.HasSuffix(paths.Config(), filepath.Join("custom", "config", "myapp")))
}

func TestXdg_XDGWithPrefix_Good(t *core.T) {
	paths := XDGWithPrefix("testing")
	core.AssertEqual(t, "testing", paths.Prefix())
	core.AssertContains(t, paths.Config(), "testing")
}

func TestXdg_DefaultHomes_Ugly(t *core.T) {
	configHome := defaultConfigHome()
	dataHome := defaultDataHome()
	core.AssertNotEmpty(t, configHome)
	core.AssertNotEmpty(t, dataHome)
}

func TestXdg_XDGWithPrefix_Bad(t *core.T) {
	paths := XDGWithPrefix("")
	got := paths.Prefix()
	core.AssertEqual(t, "core", got)
}

func TestXdg_XDGWithPrefix_Ugly(t *core.T) {
	paths := XDGWithPrefix("core tools")
	got := paths.Config()
	core.AssertContains(t, got, "core tools")
}

func TestXdg_XDGPaths_Config_Good(t *core.T) {
	paths := XDGWithPrefix("codex")
	got := paths.Config()
	core.AssertContains(t, got, "codex")
}

func TestXdg_XDGPaths_Config_Bad(t *core.T) {
	paths := XDGWithPrefix("")
	got := paths.Config()
	core.AssertContains(t, got, "core")
}

func TestXdg_XDGPaths_Config_Ugly(t *core.T) {
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(t.TempDir(), "config"))
	paths := XDGWithPrefix("codex")
	got := paths.Config()
	core.AssertTrue(t, strings.HasSuffix(got, filepath.Join("config", "codex")))
}

func TestXdg_XDGPaths_Data_Good(t *core.T) {
	paths := XDGWithPrefix("codex")
	got := paths.Data()
	core.AssertContains(t, got, "codex")
}

func TestXdg_XDGPaths_Data_Bad(t *core.T) {
	paths := XDGWithPrefix("")
	got := paths.Data()
	core.AssertContains(t, got, "core")
}

func TestXdg_XDGPaths_Data_Ugly(t *core.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(t.TempDir(), "data"))
	paths := XDGWithPrefix("codex")
	got := paths.Data()
	core.AssertTrue(t, strings.HasSuffix(got, filepath.Join("data", "codex")))
}

func TestXdg_XDGPaths_Cache_Good(t *core.T) {
	paths := XDGWithPrefix("codex")
	got := paths.Cache()
	core.AssertContains(t, got, "codex")
}

func TestXdg_XDGPaths_Cache_Bad(t *core.T) {
	paths := XDGWithPrefix("")
	got := paths.Cache()
	core.AssertContains(t, got, "core")
}

func TestXdg_XDGPaths_Cache_Ugly(t *core.T) {
	t.Setenv("XDG_CACHE_HOME", filepath.Join(t.TempDir(), "cache"))
	paths := XDGWithPrefix("codex")
	got := paths.Cache()
	core.AssertTrue(t, strings.HasSuffix(got, filepath.Join("cache", "codex")))
}

func TestXdg_XDGPaths_Runtime_Good(t *core.T) {
	paths := XDGWithPrefix("codex")
	got := paths.Runtime()
	core.AssertContains(t, got, "codex")
}

func TestXdg_XDGPaths_Runtime_Bad(t *core.T) {
	paths := XDGWithPrefix("")
	got := paths.Runtime()
	core.AssertContains(t, got, "core")
}

func TestXdg_XDGPaths_Runtime_Ugly(t *core.T) {
	t.Setenv("XDG_RUNTIME_DIR", filepath.Join(t.TempDir(), "runtime"))
	paths := XDGWithPrefix("codex")
	got := paths.Runtime()
	core.AssertTrue(t, strings.HasSuffix(got, filepath.Join("runtime", "codex")))
}

func TestXdg_XDGPaths_Prefix_Good(t *core.T) {
	paths := XDGWithPrefix("codex")
	got := paths.Prefix()
	core.AssertEqual(t, "codex", got)
}

func TestXdg_XDGPaths_Prefix_Bad(t *core.T) {
	paths := XDGWithPrefix("")
	got := paths.Prefix()
	core.AssertEqual(t, "core", got)
}

func TestXdg_XDGPaths_Prefix_Ugly(t *core.T) {
	paths := XDGWithPrefix("core tools")
	got := paths.Prefix()
	core.AssertEqual(t, "core tools", got)
}
