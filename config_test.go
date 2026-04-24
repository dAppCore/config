package config

import (
	"context"
	"fmt"
	"maps"
	"os"
	"reflect"
	"testing"

	coreio "dappco.re/go/io"
	core "dappco.re/go/core"
)

func TestConfig_Get_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	err = cfg.Set("app.name", "core")
	assertNoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	assertNoError(t, err)
	if name != "core" {
		t.Fatalf("expected name=core, got %q", name)
	}
}

func TestConfig_Get_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	var value string
	err = cfg.Get("nonexistent.key", &value)
	assertErrContains(t, err, "key not found")
}

func TestConfig_Set_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	err = cfg.Set("dev.editor", "vim")
	assertNoError(t, err)

	err = cfg.Commit()
	assertNoError(t, err)

	content, readErr := m.Read("/tmp/test/config.yaml")
	assertNoError(t, readErr)
	assertContains(t, content, "editor: vim")

	var editor string
	err = cfg.Get("dev.editor", &editor)
	assertNoError(t, err)
	if editor != "vim" {
		t.Fatalf("expected editor=vim, got %q", editor)
	}
}

func TestConfig_Set_Nested_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	err = cfg.Set("a.b.c", "deep")
	assertNoError(t, err)

	var val string
	err = cfg.Get("a.b.c", &val)
	assertNoError(t, err)
	if val != "deep" {
		t.Fatalf("expected val=deep, got %q", val)
	}
}

func TestConfig_All_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	_ = cfg.Set("key1", "val1")
	_ = cfg.Set("key2", "val2")

	all := maps.Collect(cfg.All())
	if all["key1"] != "val1" {
		t.Fatalf("expected key1=val1, got %v", all["key1"])
	}
	if all["key2"] != "val2" {
		t.Fatalf("expected key2=val2, got %v", all["key2"])
	}
}

func TestConfig_All_Order_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	_ = cfg.Set("zulu", "last")
	_ = cfg.Set("alpha", "first")

	var keys []string
	for key, _ := range cfg.All() {
		keys = append(keys, key)
	}

	want := []string{"alpha", "zulu"}
	if !reflect.DeepEqual(keys, want) {
		t.Fatalf("expected keys=%v, got %v", want, keys)
	}
}

func TestConfig_Path_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/custom/path/config.yaml"))
	assertNoError(t, err)

	if got := cfg.Path(); got != "/custom/path/config.yaml" {
		t.Fatalf("expected path=/custom/path/config.yaml, got %q", got)
	}
}

func TestConfig_Load_Existing_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "app:\n  name: existing\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	assertNoError(t, err)
	if name != "existing" {
		t.Fatalf("expected name=existing, got %q", name)
	}
}

func TestConfig_Env_Good(t *testing.T) {
	t.Setenv("CORE_CONFIG_DEV_EDITOR", "nano")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	var editor string
	err = cfg.Get("dev.editor", &editor)
	assertNoError(t, err)
	if editor != "nano" {
		t.Fatalf("expected editor=nano, got %q", editor)
	}
}

func TestConfig_Env_Overrides_File_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "dev:\n  editor: vim\n"

	t.Setenv("CORE_CONFIG_DEV_EDITOR", "nano")

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	var editor string
	err = cfg.Get("dev.editor", &editor)
	assertNoError(t, err)
	if editor != "nano" {
		t.Fatalf("expected editor=nano, got %q", editor)
	}
}

func TestConfig_Assign_Types_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "count: 42\nenabled: true\nratio: 3.14\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	var count int
	err = cfg.Get("count", &count)
	assertNoError(t, err)
	if count != 42 {
		t.Fatalf("expected count=42, got %d", count)
	}

	var enabled bool
	err = cfg.Get("enabled", &enabled)
	assertNoError(t, err)
	if !enabled {
		t.Fatalf("expected enabled=true")
	}

	var ratio float64
	err = cfg.Get("ratio", &ratio)
	assertNoError(t, err)
	assertInDelta(t, 3.14, ratio, 0.001)
}

func TestConfig_Assign_Any_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.yaml"))
	assertNoError(t, err)

	_ = cfg.Set("key", "value")

	var val any
	err = cfg.Get("key", &val)
	assertNoError(t, err)
	if val != "value" {
		t.Fatalf("expected val=value, got %v", val)
	}
}

func TestConfig_DefaultPath_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m))
	assertNoError(t, err)

	home, _ := os.UserHomeDir()
	want := home + "/.core/config.yaml"
	if got := cfg.Path(); got != want {
		t.Fatalf("expected path=%q, got %q", want, got)
	}
}

func TestLoadEnv_Good(t *testing.T) {
	t.Setenv("CORE_CONFIG_FOO_BAR", "baz")
	t.Setenv("CORE_CONFIG_SIMPLE", "value")

	result := LoadEnv("CORE_CONFIG_")
	if result["foo.bar"] != "baz" {
		t.Fatalf("expected foo.bar=baz, got %v", result["foo.bar"])
	}
	if result["simple"] != "value" {
		t.Fatalf("expected simple=value, got %v", result["simple"])
	}
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

	wantKeys := []string{"alpha", "setting"}
	wantValues := []string{"first", "secret"}
	if !reflect.DeepEqual(keys, wantKeys) {
		t.Fatalf("expected keys=%v, got %v", wantKeys, keys)
	}
	if !reflect.DeepEqual(values, wantValues) {
		t.Fatalf("expected values=%v, got %v", wantValues, values)
	}
}

func TestLoad_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	_, err := Load(m, "/nonexistent/file.yaml")
	assertErrContains(t, err, "failed to read config file")
}

func TestLoad_UnsupportedPath_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.json"] = `{"app":{"name":"core"}}`

	_, err := Load(m, "/tmp/test/config.json")
	assertErrContains(t, err, "unsupported config file type")
}

func TestLoad_InvalidYAML_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.yaml"] = "invalid: yaml: content: [[[["

	_, err := Load(m, "/tmp/test/config.yaml")
	assertErrContains(t, err, "failed to parse config file")
}

func TestConfig_LoadFile_JSON_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.json"] = `{"app":{"name":"core"}}`

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.json"))
	assertNoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	assertNoError(t, err)
	if name != "core" {
		t.Fatalf("expected name=core, got %q", name)
	}
}

func TestConfig_LoadFile_Extensionless_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config"] = "app:\n  name: core\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config"))
	assertNoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	assertNoError(t, err)
	if name != "core" {
		t.Fatalf("expected name=core, got %q", name)
	}
}

func TestConfig_LoadFile_TOML_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/test/config.toml"] = "app = { name = \"core\" }\n"

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.toml"))
	assertNoError(t, err)

	var name string
	err = cfg.Get("app.name", &name)
	assertNoError(t, err)
	if name != "core" {
		t.Fatalf("expected name=core, got %q", name)
	}
}

func TestConfig_LoadFile_Unsupported_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.txt"))
	assertNoError(t, err)

	m.Files["/tmp/test/config.txt"] = "app.name=core"
	err = cfg.LoadFile(m, "/tmp/test/config.txt")
	assertErrContains(t, err, "unsupported config file type")
}

func TestConfig_LoadFile_Unsupported_NoRead_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.txt"))
	assertNoError(t, err)

	err = cfg.LoadFile(m, "/tmp/test/config.txt")
	assertErrContains(t, err, "unsupported config file type")
}

func TestSave_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	data := map[string]any{
		"key": "value",
	}

	err := Save(m, "/tmp/test/config.yaml", data)
	assertNoError(t, err)

	content, readErr := m.Read("/tmp/test/config.yaml")
	assertNoError(t, readErr)
	assertContains(t, content, "key: value")
}

func TestSave_Extensionless_Good(t *testing.T) {
	m := coreio.NewMockMedium()

	err := Save(m, "/tmp/test/config", map[string]any{"key": "value"})
	assertNoError(t, err)

	content, readErr := m.Read("/tmp/test/config")
	assertNoError(t, readErr)
	assertContains(t, content, "key: value")
}

func TestSave_UnsupportedPath_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	err := Save(m, "/tmp/test/config.json", map[string]any{"key": "value"})
	assertErrContains(t, err, "unsupported config file type")
}

func TestConfig_Commit_UnsupportedPath_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	cfg, err := New(WithMedium(m), WithPath("/tmp/test/config.json"))
	assertNoError(t, err)

	err = cfg.Set("key", "value")
	assertNoError(t, err)

	err = cfg.Commit()
	assertErrContains(t, err, "unsupported config file type")
}

func TestConfig_LoadFile_Env(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.env"] = "FOO=bar\nBAZ=qux"

	cfg, err := New(WithMedium(m), WithPath("/config.yaml"))
	assertNoError(t, err)

	err = cfg.LoadFile(m, "/.env")
	assertNoError(t, err)

	var foo string
	err = cfg.Get("foo", &foo)
	assertNoError(t, err)
	if foo != "bar" {
		t.Fatalf("expected foo=bar, got %q", foo)
	}
}

func TestConfig_WithEnvPrefix(t *testing.T) {
	t.Setenv("MYAPP_SETTING", "secret")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithEnvPrefix("MYAPP"))
	assertNoError(t, err)

	var setting string
	err = cfg.Get("setting", &setting)
	assertNoError(t, err)
	if setting != "secret" {
		t.Fatalf("expected setting=secret, got %q", setting)
	}
}

func TestConfig_WithEnvPrefix_TrailingUnderscore_Good(t *testing.T) {
	t.Setenv("MYAPP_SETTING", "secret")

	m := coreio.NewMockMedium()
	cfg, err := New(WithMedium(m), WithEnvPrefix("MYAPP_"))
	assertNoError(t, err)

	var setting string
	err = cfg.Get("setting", &setting)
	assertNoError(t, err)
	if setting != "secret" {
		t.Fatalf("expected setting=secret, got %q", setting)
	}
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

	err := svc.OnStartup(context.Background())
	assertNoError(t, err)

	var setting string
	err = svc.Get("setting", &setting)
	assertNoError(t, err)
	if setting != "secret" {
		t.Fatalf("expected setting=secret, got %q", setting)
	}
}

func TestConfig_Get_EmptyKey(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/config.yaml"] = "app:\n  name: test\nversion: 1"

	cfg, err := New(WithMedium(m), WithPath("/config.yaml"))
	assertNoError(t, err)

	type AppConfig struct {
		App struct {
			Name string `mapstructure:"name"`
		} `mapstructure:"app"`
		Version int `mapstructure:"version"`
	}

	var full AppConfig
	err = cfg.Get("", &full)
	assertNoError(t, err)
	if full.App.Name != "test" {
		t.Fatalf("expected App.Name=test, got %q", full.App.Name)
	}
	if full.Version != 1 {
		t.Fatalf("expected Version=1, got %d", full.Version)
	}
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
