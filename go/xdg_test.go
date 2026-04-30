package config

import (
	core "dappco.re/go"
)

const (
	xdgCoreSuffixUnix    = "/core"
	xdgCoreSuffixWindows = "\\core"
	xdgCoreToolsPrefix   = "core tools"
)

func TestXdg_XDG_Good(t *core.T) {
	paths := XDG()
	core.AssertEqual(t, "core", paths.Prefix())
	core.AssertTrue(t, core.HasSuffix(paths.Config(), xdgCoreSuffixUnix) || core.HasSuffix(paths.Config(), xdgCoreSuffixWindows))
	core.AssertTrue(t, core.HasSuffix(paths.Data(), xdgCoreSuffixUnix) || core.HasSuffix(paths.Data(), xdgCoreSuffixWindows))
	core.AssertTrue(t, core.HasSuffix(paths.Cache(), xdgCoreSuffixUnix) || core.HasSuffix(paths.Cache(), xdgCoreSuffixWindows))
	core.AssertTrue(t, core.HasSuffix(paths.Runtime(), xdgCoreSuffixUnix) || core.HasSuffix(paths.Runtime(), xdgCoreSuffixWindows))
}

func TestXdg_XDG_Bad(t *core.T) {
	// An empty prefix falls back to the default "core" — no panic, no empty paths.
	defaultPaths := XDG()
	paths := XDGWithPrefix("")
	core.AssertEqual(t, defaultPaths.Prefix(), paths.Prefix())
	configPath := paths.Config()
	core.AssertEqual(t, "core", paths.Prefix())
	core.AssertContains(t, configPath, "core")
}

func TestXdg_XDG_Ugly(t *core.T) {
	// Overriding XDG_CONFIG_HOME via env must change the resolved Config dir.
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	defaultPaths := XDG()
	core.AssertContains(t, defaultPaths.Config(), "core")
	paths := XDGWithPrefix("myapp")
	core.AssertTrue(t, core.HasSuffix(paths.Config(), core.PathJoin("custom", "config", "myapp")))
}

func TestXdg_XDGWithPrefix_Good(t *core.T) {
	paths := XDGWithPrefix("testing")
	core.AssertEqual(t, "testing", paths.Prefix())
	core.AssertContains(t, paths.Config(), "testing")
}

func TestXdg_defaultConfigHome_DefaultHomes_Ugly(t *core.T) {
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
	paths := XDGWithPrefix(xdgCoreToolsPrefix)
	got := paths.Config()
	core.AssertContains(t, got, xdgCoreToolsPrefix)
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
	t.Setenv("XDG_CONFIG_HOME", core.PathJoin(t.TempDir(), "config"))
	paths := XDGWithPrefix("codex")
	got := paths.Config()
	core.AssertTrue(t, core.HasSuffix(got, core.PathJoin("config", "codex")))
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
	t.Setenv("XDG_DATA_HOME", core.PathJoin(t.TempDir(), "data"))
	paths := XDGWithPrefix("codex")
	got := paths.Data()
	core.AssertTrue(t, core.HasSuffix(got, core.PathJoin("data", "codex")))
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
	t.Setenv("XDG_CACHE_HOME", core.PathJoin(t.TempDir(), "cache"))
	paths := XDGWithPrefix("codex")
	got := paths.Cache()
	core.AssertTrue(t, core.HasSuffix(got, core.PathJoin("cache", "codex")))
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
	t.Setenv("XDG_RUNTIME_DIR", core.PathJoin(t.TempDir(), "runtime"))
	paths := XDGWithPrefix("codex")
	got := paths.Runtime()
	core.AssertTrue(t, core.HasSuffix(got, core.PathJoin("runtime", "codex")))
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
	core.AssertContains(t, paths.Config(), xdgCoreSuffixUnix)
}

func TestXdg_XDGPaths_Prefix_Ugly(t *core.T) {
	paths := XDGWithPrefix(xdgCoreToolsPrefix)
	got := paths.Prefix()
	core.AssertEqual(t, xdgCoreToolsPrefix, got)
}
