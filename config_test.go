package config

import (
	"context"
	"fmt"
	"maps"
	"io/fs"
	"os"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

func TestConfig_Get_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	err = cfg.Set("app.name", "core")
	assert.NoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	assert.NoError(t, err)
	assert.Equal(t, "core", name)
}

func TestConfig_Get_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	var value string
	err = cfg.Get("nonexistent.key", &value)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "key not found")
}

func TestConfig_Set_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	err = cfg.Set("dev.editor", "vim")
	assert.NoError(t, err)

	err = cfg.Commit()
	assert.NoError(t, err)

	// Verify the value was saved to the medium
	content, readErr := m.Read("/tmp/test/config.yaml")
	assert.NoError(t, readErr)
	assert.Contains(t, content, "editor: vim")

	// Verify we can read it back
	var editor string
	err = cfg.Get("dev.editor", &editor)
	assert.NoError(t, err)
	assert.Equal(t, "vim", editor)
}

func TestConfig_Set_Nested_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	err = cfg.Set("a.b.c", "deep")
	assert.NoError(t, err)

	var val string
	err = cfg.Get("a.b.c", &val)
	assert.NoError(t, err)
	assert.Equal(t, "deep", val)
}

func TestConfig_All_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	_ = cfg.Set("key1", "val1")
	_ = cfg.Set("key2", "val2")

	all := maps.Collect(cfg.All())
	assert.Equal(t, "val1", all["key1"])
	assert.Equal(t, "val2", all["key2"])
}

func TestConfig_All_Order_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	_ = cfg.Set("zulu", "last")
	_ = cfg.Set("alpha", "first")

	var keys []string
	for key := range cfg.All() {
		keys = append(keys, key)
	}

	assert.Equal(t, []string{"alpha", "zulu"}, keys)
}

func TestConfig_All_Nested_Good(t *testing.T) {
	// Nested keys surface via flat dot-notation — callers iterate a single
	// map instead of recursing through map[string]any trees.
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "app:\n  name: core\n  version: \"1.0\"\ndev:\n  editor: vim\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	all := maps.Collect(cfg.All())
	assert.Equal(t, "core", all["app.name"])
	assert.Equal(t, "1.0", all["app.version"])
	assert.Equal(t, "vim", all["dev.editor"])
}

func TestConfig_All_IncludesEnv_Good(t *testing.T) {
	// Env-prefixed vars that never appear in the file still surface via All()
	// so consumers iterate the merged reality, not just the persisted surface.
	t.Setenv("CORE_CONFIG_RUNTIME_TOKEN", "secret")

	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "app:\n  name: core\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	all := maps.Collect(cfg.All())
	assert.Equal(t, "core", all["app.name"])
	assert.Equal(t, "secret", all["runtime.token"])
}

func TestConfig_All_EnvOverridesFile_Good(t *testing.T) {
	// When file and env both define a key, All() reflects the env override
	// (same precedence as Get()).
	t.Setenv("CORE_CONFIG_DEV_EDITOR", "nano")

	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "dev:\n  editor: vim\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	all := maps.Collect(cfg.All())
	assert.Equal(t, "nano", all["dev.editor"])
}

func TestConfig_All_CustomPrefix_Good(t *testing.T) {
	// A custom env prefix (via WithEnvPrefix) still populates All().
	t.Setenv("MYAPP_FEATURE_BETA", "true")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"), WithEnvPrefix("MYAPP"))
	assert.NoError(t, err)

	all := maps.Collect(cfg.All())
	assert.Equal(t, "true", all["feature.beta"])
}

func TestConfig_Path_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/custom/path/config.yaml"))
	assert.NoError(t, err)

	assert.Equal(t, "/custom/path/config.yaml", cfg.Path())
}

func TestConfig_Load_Existing_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "app:\n  name: existing\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	assert.NoError(t, err)
	assert.Equal(t, "existing", name)
}

func TestConfig_Load_Existing_Schema_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "features: enabled\n"

	_, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation failed")
}

func TestConfig_Env_Good(t *testing.T) {
	// Set environment variable
	t.Setenv("CORE_CONFIG_DEV_EDITOR", "nano")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	var editor string
	err = cfg.Get("dev.editor", &editor)
	assert.NoError(t, err)
	assert.Equal(t, "nano", editor)
}

func TestConfig_Env_Overrides_File_Good(t *testing.T) {
	// Set file config
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "dev:\n  editor: vim\n"

	// Set environment override
	t.Setenv("CORE_CONFIG_DEV_EDITOR", "nano")

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	var editor string
	err = cfg.Get("dev.editor", &editor)
	assert.NoError(t, err)
	assert.Equal(t, "nano", editor)
}

func TestConfig_Assign_Types_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "count: 42\nenabled: true\nratio: 3.14\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	var count int
	err = cfg.Get("count", &count)
	assert.NoError(t, err)
	assert.Equal(t, 42, count)

	var enabled bool
	err = cfg.Get("enabled", &enabled)
	assert.NoError(t, err)
	assert.True(t, enabled)

	var ratio float64
	err = cfg.Get("ratio", &ratio)
	assert.NoError(t, err)
	assert.InDelta(t, 3.14, ratio, 0.001)
}

func TestConfig_Assign_Any_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assert.NoError(t, err)

	_ = cfg.Set("key", "value")

	var val any
	err = cfg.Get("key", &val)
	assert.NoError(t, err)
	assert.Equal(t, "value", val)
}

func TestConfig_DefaultPath_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m))
	assert.NoError(t, err)

	home, _ := os.UserHomeDir()
	assert.Equal(t, home+"/.core/config.yaml", cfg.Path())
}

func TestLoadEnv_Good(t *testing.T) {
	t.Setenv("CORE_CONFIG_FOO_BAR", "baz")
	t.Setenv("CORE_CONFIG_SIMPLE", "value")

	result := LoadEnv("CORE_CONFIG_")
	assert.Equal(t, "baz", result["foo.bar"])
	assert.Equal(t, "value", result["simple"])
}

func TestLoadEnv_PrefixNormalisation_Good(t *testing.T) {
	t.Setenv("MYAPP_SETTING", "secret")
	t.Setenv("MYAPP_ALPHA", "first")

	keys := make([]string, 0, 2)
	values := make([]string, 0, 2)
	for key, value := range Env("MYAPP") {
		keys = append(keys, key)
		values = append(values, value.(string))
	}

	assert.Equal(t, []string{"alpha", "setting"}, keys)
	assert.Equal(t, []string{"first", "secret"}, values)
}

func TestLoad_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	_, err := Load(m, "/nonexistent/file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read config file")
}

func TestLoad_UnsupportedPath_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.json"] = `{"app":{"name":"core"}}`

	_, err := Load(m, "/tmp/test/config.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config file type")
}

func TestLoad_InvalidYAML_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "invalid: yaml: content: [[[["

	_, err := Load(m, "/tmp/test/config.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse config file")
}

func TestConfig_LoadFile_JSON_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.json"] = `{"app":{"name":"core"}}`

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.json"))
	assert.NoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	assert.NoError(t, err)
	assert.Equal(t, "core", name)
}

func TestConfig_LoadFile_Extensionless_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config"] = "app:\n  name: core\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config"))
	assert.NoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	assert.NoError(t, err)
	assert.Equal(t, "core", name)
}

func TestConfig_LoadFile_TOML_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.toml"] = "app = { name = \"core\" }\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.toml"))
	assert.NoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	assert.NoError(t, err)
	assert.Equal(t, "core", name)
}

func TestConfig_LoadFile_Unsupported_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.txt"))
	assert.NoError(t, err)

	m.Files["/tmp/test/config.txt"] = "app.name=core"
	err = cfg.LoadFile(m, "/tmp/test/config.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config file type")
}

func TestConfig_LoadFile_Unsupported_NoRead_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.txt"))
	assert.NoError(t, err)

	err = cfg.LoadFile(m, "/tmp/test/config.txt")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config file type")
}

func TestSave_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	data := map[string]any{
		"key": "value",
	}

	err := Save(m, "/tmp/test/config.yaml", data)
	assert.NoError(t, err)

	content, readErr := m.Read("/tmp/test/config.yaml")
	assert.NoError(t, readErr)
	assert.Contains(t, content, "key: value")

	info, statErr := m.Stat("/tmp/test/config.yaml")
	assert.NoError(t, statErr)
	assert.Equal(t, fs.FileMode(0600), info.Mode())
}

func TestSave_Extensionless_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	err := Save(m, "/tmp/test/config", map[string]any{"key": "value"})
	assert.NoError(t, err)

	content, readErr := m.Read("/tmp/test/config")
	assert.NoError(t, readErr)
	assert.Contains(t, content, "key: value")
}

func TestSave_UnsupportedPath_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	err := Save(m, "/tmp/test/config.json", map[string]any{"key": "value"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config file type")
}

func TestConfig_Commit_UnsupportedPath_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.json"))
	assert.NoError(t, err)

	err = cfg.Set("key", "value")
	assert.NoError(t, err)

	err = cfg.Commit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported config file type")
}

func TestConfig_LoadFile_Env_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.env"] = "FOO=bar\nBAZ=qux"

	cfg, err := New(WithMedium(m), WithPath("/config.yaml"))
	assert.NoError(t, err)

	err = cfg.LoadFile(m, "/.env")
	assert.NoError(t, err)

	var foo string
	err = cfg.Get("foo", &foo)
	assert.NoError(t, err)
	assert.Equal(t, "bar", foo)
}

func TestConfig_WithEnvPrefix_Good(t *testing.T) {
	t.Setenv("MYAPP_SETTING", "secret")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithEnvPrefix("MYAPP"))
	assert.NoError(t, err)

	var setting string
	err = cfg.Get("setting", &setting)
	assert.NoError(t, err)
	assert.Equal(t, "secret", setting)
}

func TestConfig_WithEnvPrefix_TrailingUnderscore_Good(t *testing.T) {
	t.Setenv("MYAPP_SETTING", "secret")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithEnvPrefix("MYAPP_"))
	assert.NoError(t, err)

	var setting string
	err = cfg.Get("setting", &setting)
	assert.NoError(t, err)
	assert.Equal(t, "secret", setting)
}

func TestService_OnStartup_WithEnvPrefix_Good(t *testing.T) {
	t.Setenv("MYAPP_SETTING", "secret")

	m := coreio.NewMockMedium()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(nil, ServiceOptions{
			EnvPrefix: "MYAPP",
			Medium:    m,
		}),
	}

	result := svc.OnStartup(context.Background())
	assert.True(t, result.OK)

	var setting string
	err := svc.Get("setting", &setting)
	assert.NoError(t, err)
	assert.Equal(t, "secret", setting)
}

func TestConfig_Get_EmptyKey_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/config.yaml"] = "app:\n  name: test\nversion: 1"

	cfg, err := New(WithMedium(m), WithPath("/config.yaml"))
	assert.NoError(t, err)

	type AppConfig struct {
		App struct {
			Name string `mapstructure:"name"`
		} `mapstructure:"app"`
		Version int `mapstructure:"version"`
	}

	var full AppConfig
	err = cfg.Get("", &full)
	assert.NoError(t, err)
	assert.Equal(t, "test", full.App.Name)
	assert.Equal(t, 1, full.Version)
}

func ExampleConfig_Get() {
	m := coreio.NewMockMedium()

	cfg, _ := New(WithMedium(m), WithPath("/tmp/example/config.yaml"))
	_ = cfg.Set("dev.editor", "vim")

	var editor string
	_ = cfg.Get("dev.editor", &editor)

	fmt.Println(editor)
	// Output: vim
}

func ExampleConfig_Commit() {
	m := coreio.NewMockMedium()

	cfg, _ := New(WithMedium(m), WithPath("/tmp/example/config.yaml"))
	_ = cfg.Set("app.name", "core")
	_ = cfg.Commit()

	content, _ := m.Read("/tmp/example/config.yaml")
	fmt.Print(content)
	// Output:
	// app:
	//     name: core
	// version: 1
}

func ExampleEnv() {
	t := "EXAMPLE_FOO_BAR"
	_ = os.Setenv(t, "baz")
	defer os.Unsetenv(t)

	for key, value := range Env("EXAMPLE_") {
		fmt.Printf("%s=%s\n", key, value)
	}

	// Output: foo.bar=baz
}

func ExampleConfig_LoadFile() {
	m := coreio.NewMockMedium()
	m.Files["/.env"] = "FOO=bar\n"

	cfg, _ := New(WithMedium(m), WithPath("/config.yaml"))
	_ = cfg.LoadFile(m, "/.env")

	var foo string
	_ = cfg.Get("foo", &foo)

	fmt.Println(foo)
	// Output: bar
}
