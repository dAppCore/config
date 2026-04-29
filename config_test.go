package config

import (
	"context"
	"io/fs"
	"iter"
	"maps"
	"syscall"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func requireResultOK(t *core.T, r core.Result) {
	t.Helper()
	if !r.OK {
		core.RequireNoError(t, resultError(r))
	}
}

func testMkdirAll(t *core.T, path string, mode core.FileMode) {
	t.Helper()
	requireResultOK(t, core.MkdirAll(path, mode))
}

func testWriteFile(t *core.T, path string, data []byte, mode core.FileMode) {
	t.Helper()
	requireResultOK(t, core.WriteFile(path, data, mode))
}

func testSymlink(t *core.T, target, link string) {
	t.Helper()
	core.RequireNoError(t, syscall.Symlink(target, link))
}

func testRemove(path string) {
	_ = core.Remove(path)
}

func testGetwd(t *core.T) string {
	t.Helper()
	r := core.Getwd()
	requireResultOK(t, r)
	return r.Value.(string)
}

func testChdir(t *core.T, dir string) {
	t.Helper()
	requireResultOK(t, core.Chdir(dir))
}

func testPathAbs(t *core.T, path string) string {
	t.Helper()
	r := core.PathAbs(path)
	requireResultOK(t, r)
	return r.Value.(string)
}

func testPathEvalSymlinks(t *core.T, path string) string {
	t.Helper()
	r := core.PathEvalSymlinks(path)
	requireResultOK(t, r)
	return r.Value.(string)
}

func TestConfig_Get_Good(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	err = cfg.Set("app.name", "core")
	core.AssertNoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", name)
}

func TestConfig_Get_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	var value string
	err = cfg.Get("nonexistent.key", &value)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "key not found")
}

func TestConfig_Set_Good(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	err = cfg.Set("dev.editor", "vim")
	core.AssertNoError(t, err)

	err = cfg.Commit()
	core.AssertNoError(t, err)

	// Verify the value was saved to the medium
	content, readErr := m.Read("/tmp/test/config.yaml")
	core.AssertNoError(t, readErr)
	core.AssertContains(t, content, "editor: vim")

	// Verify we can read it back
	var editor string
	err = cfg.Get("dev.editor", &editor)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "vim", editor)
}

func TestConfig_Set_Nested_Good(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	err = cfg.Set("a.b.c", "deep")
	core.AssertNoError(t, err)

	var val string
	err = cfg.Get("a.b.c", &val)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "deep", val)
}

func TestConfig_All_Good(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	_ = cfg.Set("key1", "val1")
	_ = cfg.Set("key2", "val2")

	all := maps.Collect(cfg.All())
	core.AssertEqual(t, "val1", all["key1"])
	core.AssertEqual(t, "val2", all["key2"])
}

func TestConfig_All_Order_Good(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	_ = cfg.Set("zulu", "last")
	_ = cfg.Set("alpha", "first")

	var keys []string
	for key := range cfg.All() {
		keys = append(keys, key)
	}

	core.AssertEqual(t, []string{"alpha", "zulu"}, keys)
}

func TestConfig_All_Snapshot_Good(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	_ = cfg.Set("alpha", "one")
	snapshot := cfg.All()
	_ = cfg.Set("beta", "two")

	all := maps.Collect(snapshot)
	core.AssertEqual(t, "one", all["alpha"])
	core.AssertNotContains(t, all, "beta")
}

func TestConfig_All_Nested_Good(t *core.T) {
	// Nested keys surface via flat dot-notation — callers iterate a single
	// map instead of recursing through map[string]any trees.
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "app:\n  name: core\n  version: \"1.0\"\ndev:\n  editor: vim\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	all := maps.Collect(cfg.All())
	core.AssertEqual(t, "core", all["app.name"])
	core.AssertEqual(t, "1.0", all["app.version"])
	core.AssertEqual(t, "vim", all["dev.editor"])
}

func TestConfig_All_IncludesEnv_Good(t *core.T) {
	// Env-prefixed vars that never appear in the file still surface via All()
	// so consumers iterate the merged reality, not just the persisted surface.
	t.Setenv("CORE_CONFIG_RUNTIME_TOKEN", "secret")

	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "app:\n  name: core\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	all := maps.Collect(cfg.All())
	core.AssertEqual(t, "core", all["app.name"])
	core.AssertEqual(t, "secret", all["runtime.token"])
}

func TestConfig_All_EnvOverridesFile_Good(t *core.T) {
	// When file and env both define a key, All() reflects the env override
	// (same precedence as Get()).
	t.Setenv("CORE_CONFIG_DEV_EDITOR", "nano")

	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "dev:\n  editor: vim\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	all := maps.Collect(cfg.All())
	core.AssertEqual(t, "nano", all["dev.editor"])
}

func TestConfig_All_CustomPrefix_Good(t *core.T) {
	// A custom env prefix (via WithEnvPrefix) still populates All().
	t.Setenv("MYAPP_FEATURE_BETA", "true")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"), WithEnvPrefix("MYAPP"))
	core.AssertNoError(t, err)

	all := maps.Collect(cfg.All())
	core.AssertEqual(t, "true", all["feature.beta"])
}

func TestConfig_Path_Good(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/custom/path/config.yaml"))
	core.AssertNoError(t, err)

	core.AssertEqual(t, "/custom/path/config.yaml", cfg.Path())
}

func TestConfigLoadExistingGood(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "app:\n  name: existing\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "existing", name)
}

func TestConfigLoadExistingSchemaBad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "features: enabled\n"

	_, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "schema validation failed")
}

func TestConfigEnvGood(t *core.T) {
	// Set environment variable
	t.Setenv("CORE_CONFIG_DEV_EDITOR", "nano")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	var editor string
	err = cfg.Get("dev.editor", &editor)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "nano", editor)
}

func TestConfigEnvOverridesFileGood(t *core.T) {
	// Set file config
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "dev:\n  editor: vim\n"

	// Set environment override
	t.Setenv("CORE_CONFIG_DEV_EDITOR", "nano")

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	var editor string
	err = cfg.Get("dev.editor", &editor)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "nano", editor)
}

func TestConfigAssignTypesGood(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "count: 42\nenabled: true\nratio: 3.14\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	var count int
	err = cfg.Get("count", &count)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 42, count)

	var enabled bool
	err = cfg.Get("enabled", &enabled)
	core.AssertNoError(t, err)
	core.AssertTrue(t, enabled)

	var ratio float64
	err = cfg.Get("ratio", &ratio)
	core.AssertNoError(t, err)
	core.AssertInDelta(t, 3.14, ratio, 0.001)
}

func TestConfigAssignAnyGood(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	core.AssertNoError(t, err)

	_ = cfg.Set("key", "value")

	var val any
	err = cfg.Get("key", &val)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "value", val)
}

func TestConfigDefaultPathGood(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m))
	core.AssertNoError(t, err)

	home := core.UserHomeDir().Value.(string)
	core.AssertEqual(t, home+"/.core/config.yaml", cfg.Path())
}

func TestConfig_New_NoHome_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	core.RequireNoError(t, m.Write("/tmp/nohome.yaml", "bad: [yaml"))
	cfg, err := New(WithMedium(m), WithPath("/tmp/nohome.yaml"))
	core.AssertNil(t, cfg)
	core.AssertError(t, err)
}

func TestLoadEnv_Good(t *core.T) {
	t.Setenv("CORE_CONFIG_FOO_BAR", "baz")
	t.Setenv("CORE_CONFIG_SIMPLE", "value")

	result := LoadEnv("CORE_CONFIG_")
	core.AssertEqual(t, "baz", result["foo.bar"])
	core.AssertEqual(t, "value", result["simple"])
}

func TestLoadEnvPrefixNormalisationGood(t *core.T) {
	t.Setenv("MYAPP_SETTING", "secret")
	t.Setenv("MYAPP_ALPHA", "first")

	keys := make([]string, 0, 2)
	values := make([]string, 0, 2)
	for key, value := range Env("MYAPP") {
		keys = append(keys, key)
		values = append(values, value.(string))
	}

	core.AssertEqual(t, []string{"alpha", "setting"}, keys)
	core.AssertEqual(t, []string{"first", "secret"}, values)
}

func TestLoad_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	_, err := Load(m, "/nonexistent/file.yaml")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to read config file")
}

func TestLoadUnsupportedPathBad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.json"] = `{"app":{"name":"core"}}`

	_, err := Load(m, "/tmp/test/config.json")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsupported config file type")
}

func TestLoadInvalidYAMLBad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "invalid: yaml: content: [[[["

	_, err := Load(m, "/tmp/test/config.yaml")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to parse config file")
}

func TestConfigLoadFileJSONGood(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.json"] = `{"app":{"name":"core"}}`

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.json"))
	core.AssertNoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", name)
}

func TestConfigLoadFileExtensionlessGood(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config"] = "app:\n  name: core\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config"))
	core.AssertNoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", name)
}

func TestConfigLoadFileTOMLGood(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.toml"] = "app = { name = \"core\" }\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.toml"))
	core.AssertNoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", name)
}

func TestConfig_LoadFile_Unsupported_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.txt"))
	core.AssertNoError(t, err)

	m.Files["/tmp/test/config.txt"] = "app.name=core"
	err = cfg.LoadFile(m, "/tmp/test/config.txt")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsupported config file type")
}

func TestConfig_LoadFile_Unsupported_NoRead_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.txt"))
	core.AssertNoError(t, err)

	err = cfg.LoadFile(m, "/tmp/test/config.txt")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsupported config file type")
}

func TestSave_Good(t *core.T) {
	m := coreio.NewMockMedium()

	data := map[string]any{
		"key": "value",
	}

	err := Save(m, "/tmp/test/config.yaml", data)
	core.AssertNoError(t, err)

	content, readErr := m.Read("/tmp/test/config.yaml")
	core.AssertNoError(t, readErr)
	core.AssertContains(t, content, "key: value")
	core.AssertContains(t, content, "version: 1")

	info, statErr := m.Stat("/tmp/test/config.yaml")
	core.AssertNoError(t, statErr)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode())
}

func TestSaveExtensionlessGood(t *core.T) {
	m := coreio.NewMockMedium()

	err := Save(m, "/tmp/test/config", map[string]any{"key": "value"})
	core.AssertNoError(t, err)

	content, readErr := m.Read("/tmp/test/config")
	core.AssertNoError(t, readErr)
	core.AssertContains(t, content, "key: value")
}

func TestSaveUnsupportedPathBad(t *core.T) {
	m := coreio.NewMockMedium()

	err := Save(m, "/tmp/test/config.json", map[string]any{"key": "value"})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsupported config file type")
}

func TestConfig_Commit_UnsupportedPath_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.json"))
	core.AssertNoError(t, err)

	err = cfg.Set("key", "value")
	core.AssertNoError(t, err)

	err = cfg.Commit()
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsupported config file type")
}

func TestConfig_LoadFile_Env_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/.env"] = "FOO=bar\nBAZ=qux"

	cfg, err := New(WithMedium(m), WithPath("/config.yaml"))
	core.AssertNoError(t, err)

	err = cfg.LoadFile(m, "/.env")
	core.AssertNoError(t, err)

	var foo string
	err = cfg.Get("foo", &foo)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "bar", foo)
}

func TestConfig_WithEnvPrefix_Good(t *core.T) {
	t.Setenv("MYAPP_SETTING", "secret")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithEnvPrefix("MYAPP"))
	core.AssertNoError(t, err)

	var setting string
	err = cfg.Get("setting", &setting)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "secret", setting)
}

func TestConfig_WithEnvPrefix_TrailingUnderscore_Good(t *core.T) {
	t.Setenv("MYAPP_SETTING", "secret")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithEnvPrefix("MYAPP_"))
	core.AssertNoError(t, err)

	var setting string
	err = cfg.Get("setting", &setting)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "secret", setting)
}

func TestService_OnStartup_WithEnvPrefix_Good(t *core.T) {
	t.Setenv("MYAPP_SETTING", "secret")

	m := coreio.NewMockMedium()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(nil, ServiceOptions{
			EnvPrefix: "MYAPP",
			Medium:    m,
		}),
	}

	result := svc.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)

	var setting string
	err := svc.Get("setting", &setting)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "secret", setting)
}

func TestConfig_Get_EmptyKey_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/config.yaml"] = "app:\n  name: test\nversion: 1"

	cfg, err := New(WithMedium(m), WithPath("/config.yaml"))
	core.AssertNoError(t, err)

	type AppConfig struct {
		App struct {
			Name string `mapstructure:"name"`
		} `mapstructure:"app"`
		Version int `mapstructure:"version"`
	}

	var full AppConfig
	err = cfg.Get("", &full)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "test", full.App.Name)
	core.AssertEqual(t, 1, full.Version)
}

func axConfigFixture(t *core.T) (*Config, *coreio.MockMedium, string) {
	t.Helper()
	m := coreio.NewMockMedium()
	path := "/ax7/config.yaml"
	cfg, err := New(WithMedium(m), WithPath(path))
	core.RequireNoError(t, err)
	return cfg, m, path
}

func TestConfig_WithMedium_Good(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/ax7/medium.yaml"))
	core.RequireNoError(t, err)
	core.AssertSame(t, m, cfg.Medium())
}

func TestConfig_WithMedium_Bad(t *core.T) {
	cfg, err := New(WithMedium(nil), WithPath("/ax7/medium.yaml"))
	core.RequireNoError(t, err)
	core.AssertNotNil(t, cfg.Medium())
}

func TestConfig_WithMedium_Ugly(t *core.T) {
	first := coreio.NewMockMedium()
	second := coreio.NewMockMedium()
	cfg, err := New(WithMedium(first), WithMedium(second), WithPath("/ax7/medium.yaml"))
	core.RequireNoError(t, err)
	core.AssertSame(t, second, cfg.Medium())
}

func TestConfig_WithPath_Good(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/ax7/path.yaml"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "/ax7/path.yaml", cfg.Path())
}

func TestConfig_WithPath_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath(""))
	core.RequireNoError(t, err)
	core.AssertContains(t, cfg.Path(), ".core/config.yaml")
}

func TestConfig_WithPath_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/ax7/first.yaml"), WithPath("/ax7/second.yaml"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "/ax7/second.yaml", cfg.Path())
}

func TestConfig_WithEnvPrefix_Bad(t *core.T) {
	t.Setenv("AX7_TRAIL_NAME", "trail")
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/env.yaml"), WithEnvPrefix("AX7_TRAIL_"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "trail", mapFromSeq(cfg.All())["name"])
}

func TestConfig_WithEnvPrefix_Ugly(t *core.T) {
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/env.yaml"), WithEnvPrefix(""))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "CORE_CONFIG_", envPrefixOf(cfg.full))
}

func TestConfig_WithCore_Good(t *core.T) {
	c := core.New()
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/core.yaml"), WithCore(c))
	core.RequireNoError(t, err)
	core.AssertSame(t, c, cfg.core)
}

func TestConfig_WithCore_Bad(t *core.T) {
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/core.yaml"), WithCore(nil))
	core.RequireNoError(t, err)
	core.AssertNil(t, cfg.core)
}

func TestConfig_WithCore_Ugly(t *core.T) {
	first := core.New()
	second := core.New()
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/core.yaml"), WithCore(first), WithCore(second))
	core.RequireNoError(t, err)
	core.AssertSame(t, second, cfg.core)
}

func TestConfig_WithStore_Good(t *core.T) {
	store := &mockConfigStore{}
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/store.yaml"), WithStore(store))
	core.RequireNoError(t, err)
	core.AssertSame(t, store, cfg.store)
}

func TestConfig_WithStore_Bad(t *core.T) {
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/store.yaml"), WithStore(nil))
	core.RequireNoError(t, err)
	core.AssertNil(t, cfg.store)
}

func TestConfig_WithStore_Ugly(t *core.T) {
	store := &mockConfigStore{failWith: core.NewError("store refused")}
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/store.yaml"), WithStore(store))
	core.RequireNoError(t, err)
	core.AssertNoError(t, cfg.Set("agent", "codex"))
}

func TestConfig_WithDefaults_Good(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(
		WithMedium(m),
		WithPath("/defaults.yaml"),
		WithDefaults(map[string]any{
			"dev.editor":  "vim",
			"app.version": "0.1.0",
		}),
	)
	core.AssertNoError(t, err)

	var editor, version string
	core.AssertNoError(t, cfg.Get("dev.editor", &editor))
	core.AssertEqual(t, "vim", editor)
	core.AssertNoError(t, cfg.Get("app.version", &version))
	core.AssertEqual(t, "0.1.0", version)
}

func TestConfig_WithDefaults_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	cfg, err := New(
		WithMedium(m),
		WithPath("/defaults.yaml"),
		WithDefaults(map[string]any{"dev.editor": "vim"}),
	)
	core.AssertNoError(t, err)
	core.AssertNoError(t, cfg.Set("dev.editor", "nano"))

	var editor string
	core.AssertNoError(t, cfg.Get("dev.editor", &editor))
	core.AssertEqual(t, "nano", editor)
}

func TestConfig_WithDefaults_Ugly(t *core.T) {
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/defaults.yaml"), WithDefaults(nil))
	core.RequireNoError(t, err)
	core.AssertEmpty(t, mapFromSeq(cfg.All()))
}

func TestConfig_Config_AttachCore_Good(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	c := core.New()
	cfg.AttachCore(c)
	core.AssertSame(t, c, cfg.core)
}

func TestConfig_Config_AttachCore_Bad(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	cfg.AttachCore(nil)
	core.AssertNil(t, cfg.core)
}

func TestConfig_Config_AttachCore_Ugly(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	first := core.New()
	second := core.New()
	cfg.AttachCore(first)
	cfg.AttachCore(second)
	core.AssertSame(t, second, cfg.core)
}

func TestConfig_New_Good(t *core.T) {
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/new.yaml"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "/ax7/new.yaml", cfg.Path())
}

func TestConfig_New_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	core.RequireNoError(t, m.Write("/ax7/new.yaml", "bad: [yaml"))
	cfg, err := New(WithMedium(m), WithPath("/ax7/new.yaml"))
	core.AssertNil(t, cfg)
	core.AssertError(t, err)
}

func TestConfig_New_Ugly(t *core.T) {
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/new.yaml"), WithEnvPrefix("AX7__"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "AX7_", envPrefixOf(cfg.full))
}

func TestConfig_Config_LoadFile_Good(t *core.T) {
	cfg, m, _ := axConfigFixture(t)
	core.RequireNoError(t, m.Write("/ax7/load.yaml", "app:\n  name: loaded\n"))
	err := cfg.LoadFile(m, "/ax7/load.yaml")
	core.AssertNoError(t, err)
}

func TestConfig_Config_LoadFile_Bad(t *core.T) {
	cfg, m, _ := axConfigFixture(t)
	err := cfg.LoadFile(m, "/ax7/missing.yaml")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to read config file")
}

func TestConfig_Config_LoadFile_Ugly(t *core.T) {
	cfg, m, _ := axConfigFixture(t)
	core.RequireNoError(t, m.Write("/ax7/load.unsupported", "app: config\n"))
	err := cfg.LoadFile(m, "/ax7/load.unsupported")
	core.AssertError(t, err)
}

func TestConfig_Config_Get_Good(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	core.RequireNoError(t, cfg.Set("app.name", "core"))
	var got string
	core.AssertNoError(t, cfg.Get("app.name", &got))
	core.AssertEqual(t, "core", got)
}

func TestConfig_Config_Get_Bad(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	var got string
	err := cfg.Get("missing", &got)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "key not found")
}

func TestConfig_Config_Get_Ugly(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	core.RequireNoError(t, cfg.Set("app.name", "core"))
	var got map[string]any
	core.AssertNoError(t, cfg.Get("", &got))
	core.AssertEqual(t, "core", got["app"].(map[string]any)["name"])
}

func TestConfig_Config_SetDefault_Good(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	cfg.SetDefault("app.name", "default")
	var got string
	core.AssertNoError(t, cfg.Get("app.name", &got))
	core.AssertEqual(t, "default", got)
}

func TestConfig_Config_SetDefault_Bad(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	cfg.SetDefault("app.name", "default")
	core.RequireNoError(t, cfg.Set("app.name", "set"))
	var got string
	core.RequireNoError(t, cfg.Get("app.name", &got))
	core.AssertEqual(t, "set", got)
}

func TestConfig_Config_SetDefault_Ugly(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	cfg.SetDefault("", "root-default")
	got := mapFromSeq(cfg.All())
	core.AssertEqual(t, "root-default", got[""])
}

func TestConfig_Config_Set_Good(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	err := cfg.Set("agent.name", "codex")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "codex", mapFromSeq(cfg.All())["agent.name"])
}

func TestConfig_Config_Set_Bad(t *core.T) {
	store := &mockConfigStore{failWith: core.NewError("store refused")}
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/set.yaml"), WithStore(store))
	core.RequireNoError(t, err)
	core.AssertNoError(t, cfg.Set("agent.name", "codex"))
	core.AssertEqual(t, 1, store.calls)
}

func TestConfig_Config_Set_Ugly(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	core.AssertNoError(t, cfg.Set("", "root"))
	core.AssertEqual(t, "root", cfg.file.Get(""))
}

func TestConfig_Config_Commit_Good(t *core.T) {
	cfg, m, path := axConfigFixture(t)
	core.RequireNoError(t, cfg.Set("agent", "codex"))
	err := cfg.Commit()
	core.AssertNoError(t, err)
	core.AssertTrue(t, m.Exists(path))
}

func TestConfig_Config_Commit_Bad(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	cfg.path = "/ax7/config.json"
	err := cfg.Commit()
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsupported config file type")
}

func TestConfig_Config_Commit_Ugly(t *core.T) {
	cfg, m, path := axConfigFixture(t)
	err := cfg.Commit()
	core.AssertNoError(t, err)
	core.AssertTrue(t, m.Exists(path))
}

func TestConfig_Config_All_Good(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	core.RequireNoError(t, cfg.Set("b.key", "second"))
	core.RequireNoError(t, cfg.Set("a.key", "first"))
	core.AssertEqual(t, []string{"a.key", "b.key"}, keysFromSeq(cfg.All()))
}

func TestConfig_Config_All_Bad(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	got := mapFromSeq(cfg.All())
	core.AssertEmpty(t, got)
}

func TestConfig_Config_All_Ugly(t *core.T) {
	t.Setenv("AX7_ALL_DYNAMIC", "env")
	cfg, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/all.yaml"), WithEnvPrefix("AX7_ALL"))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "env", mapFromSeq(cfg.All())["dynamic"])
}

func TestConfig_Config_Path_Good(t *core.T) {
	cfg, _, path := axConfigFixture(t)
	got := cfg.Path()
	core.AssertEqual(t, path, got)
}

func TestConfig_Config_Path_Bad(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	cfg.path = ""
	core.AssertEqual(t, "", cfg.Path())
}

func TestConfig_Config_Path_Ugly(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	cfg.path = "/tmp/../tmp/config.yaml"
	core.AssertContains(t, cfg.Path(), "../")
}

func TestConfig_Config_Medium_Good(t *core.T) {
	cfg, m, _ := axConfigFixture(t)
	got := cfg.Medium()
	core.AssertSame(t, m, got)
}

func TestConfig_Config_Medium_Bad(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	cfg.medium = nil
	got := cfg.Medium()
	core.AssertNil(t, got)
}

func TestConfig_Config_Medium_Ugly(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	replacement := coreio.NewMockMedium()
	cfg.medium = replacement
	core.AssertSame(t, replacement, cfg.Medium())
}

func TestConfig_Config_MergeFrom_Good(t *core.T) {
	target, _, _ := axConfigFixture(t)
	source, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/source.yaml"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, source.Set("dev.editor", "vim"))
	target.MergeFrom(source)
	core.AssertEqual(t, "vim", mapFromSeq(target.All())["dev.editor"])
}

func TestConfig_Config_MergeFrom_Bad(t *core.T) {
	target, _, _ := axConfigFixture(t)
	target.MergeFrom(nil)
	core.AssertEmpty(t, mapFromSeq(target.All()))
}

func TestConfig_Config_MergeFrom_Ugly(t *core.T) {
	target, _, _ := axConfigFixture(t)
	source, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/source.yaml"))
	core.RequireNoError(t, err)
	core.RequireNoError(t, target.Set("dev.editor", "emacs"))
	core.RequireNoError(t, source.Set("dev.editor", "vim"))
	target.MergeFrom(source)
	core.AssertEqual(t, "emacs", mapFromSeq(target.All())["dev.editor"])
}

func TestConfig_Config_OnChange_Good(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	seen := map[string]any{}
	cfg.OnChange(func(key string, value any) { seen[key] = value })
	core.RequireNoError(t, cfg.Set("dev.editor", "vim"))
	core.AssertEqual(t, "vim", seen["dev.editor"])
}

func TestConfig_Config_OnChange_Bad(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	cfg.OnChange(nil)
	core.RequireNoError(t, cfg.Set("dev.editor", "vim"))
	core.AssertEqual(t, "vim", mapFromSeq(cfg.All())["dev.editor"])
}

func TestConfig_Config_OnChange_Ugly(t *core.T) {
	cfg, _, _ := axConfigFixture(t)
	count := 0
	cfg.OnChange(func(string, any) { count++ })
	cfg.OnChange(func(string, any) { count++ })
	core.RequireNoError(t, cfg.Set("dev.editor", "vim"))
	core.AssertEqual(t, 2, count)
}

func TestConfig_Load_Good(t *core.T) {
	m := coreio.NewMockMedium()
	core.RequireNoError(t, m.Write("/ax7/load.yaml", "app:\n  name: core\n"))
	got, err := Load(m, "/ax7/load.yaml")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", got["app"].(map[string]any)["name"])
}

func TestConfig_Load_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := Load(m, "/ax7/missing.yaml")
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestConfig_Load_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	core.RequireNoError(t, m.Write("/ax7/.env", "FOO=bar\n"))
	got, err := Load(m, "/ax7/.env")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "bar", got["foo"])
}

func TestConfig_Save_Good(t *core.T) {
	m := coreio.NewMockMedium()
	err := Save(m, "/ax7/save.yaml", map[string]any{"app": map[string]any{"name": "core"}})
	core.AssertNoError(t, err)
	core.AssertTrue(t, m.Exists("/ax7/save.yaml"))
}

func TestConfig_Save_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	err := Save(m, "/ax7/save.json", map[string]any{"app": "core"})
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsupported config file type")
}

func TestConfig_Save_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	err := Save(m, "/ax7/save", nil)
	core.AssertNoError(t, err)
	core.AssertTrue(t, m.Exists("/ax7/save"))
}

func mapFromSeq(seq iter.Seq2[string, any]) map[string]any {
	out := map[string]any{}
	for key, value := range seq {
		out[key] = value
	}
	return out
}

func keysFromSeq(seq iter.Seq2[string, any]) []string {
	var keys []string
	for key := range seq {
		keys = append(keys, key)
	}
	return keys
}
