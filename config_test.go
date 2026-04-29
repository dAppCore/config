package config_test

import (
	. "dappco.re/go"
	config "dappco.re/go/config"
)

type refusingMedium struct{}

func (refusingMedium) Exists(string) bool { return true }

func (refusingMedium) Read(string) Result {
	return Fail(NewError("read refused"))
}

func (refusingMedium) Write(string, string) Result {
	return Fail(NewError("write refused"))
}

func (refusingMedium) EnsureDir(string) Result {
	return Fail(NewError("mkdir refused"))
}

func configTestFS(t *T) *Fs {
	t.Helper()
	root := PathEvalSymlinks(t.TempDir())
	RequireTrue(t, root.OK, root.Error())
	return (&Fs{}).New(root.Value.(string))
}

func configTestMedium(t *T) (*Fs, string) {
	t.Helper()
	return configTestFS(t), "/config.yaml"
}

func writeConfigFile(t *T, fs *Fs, path, content string) {
	t.Helper()
	r := fs.Write(path, content)
	RequireTrue(t, r.OK, r.Error())
}

func requireConfig(t *T, r Result) *config.Config {
	t.Helper()
	RequireTrue(t, r.OK, r.Error())
	cfg, ok := r.Value.(*config.Config)
	RequireTrue(t, ok)
	return cfg
}

func startedService(t *T, opts config.ServiceOptions) *config.Service {
	t.Helper()
	svc := &config.Service{ServiceRuntime: NewServiceRuntime(nil, opts)}
	r := svc.OnStartup(Background())
	RequireTrue(t, r.OK, r.Error())
	return svc
}

func TestConfig_New_Good(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, path, "app:\n  name: core\n")

	r := config.New(config.WithMedium(fs), config.WithPath(path))

	AssertTrue(t, r.OK, r.Error())
	cfg := r.Value.(*config.Config)
	var name string
	get := cfg.Get("app.name", &name)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "core", name)
}

func TestConfig_New_Bad(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.txt", "app.name=core")

	r := config.New(config.WithMedium(fs), config.WithPath("/config.txt"))

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "unsupported config file type")
}

func TestConfig_New_Ugly(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/.env", "FOO=bar\n")

	r := config.New(config.WithMedium(fs), config.WithPath("/.env"))

	AssertTrue(t, r.OK, r.Error())
	cfg := r.Value.(*config.Config)
	var foo string
	get := cfg.Get("foo", &foo)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "bar", foo)
}

func TestConfig_WithMedium_Good(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, path, "agent: codex\n")

	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))

	var agent string
	r := cfg.Get("agent", &agent)
	AssertTrue(t, r.OK, r.Error())
	AssertEqual(t, "codex", agent)
}

func TestConfig_WithMedium_Bad(t *T) {
	r := config.New(config.WithMedium(nil), config.WithPath("/missing.yaml"))

	AssertTrue(t, r.OK, r.Error())
	cfg := r.Value.(*config.Config)
	AssertEqual(t, "/missing.yaml", cfg.Path())
}

func TestConfig_WithMedium_Ugly(t *T) {
	r := config.New(config.WithMedium(refusingMedium{}), config.WithPath("/config.yaml"))

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "read refused")
}

func TestConfig_WithPath_Good(t *T) {
	fs := configTestFS(t)

	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath("/custom/config.yaml")))

	AssertEqual(t, "/custom/config.yaml", cfg.Path())
}

func TestConfig_WithPath_Bad(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/custom/config.json", `{"app":{"name":"core"}}`)

	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath("/custom/config.json")))

	AssertEqual(t, "/custom/config.json", cfg.Path())
}

func TestConfig_WithPath_Ugly(t *T) {
	fs := configTestFS(t)

	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath("")))

	AssertContains(t, cfg.Path(), ".core/config.yaml")
}

func TestConfig_WithEnvPrefix_Good(t *T) {
	t.Setenv("MYAPP_SETTING", "secret")
	fs, path := configTestMedium(t)

	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path), config.WithEnvPrefix("MYAPP")))

	var setting string
	r := cfg.Get("setting", &setting)
	AssertTrue(t, r.OK, r.Error())
	AssertEqual(t, "secret", setting)
}

func TestConfig_WithEnvPrefix_Bad(t *T) {
	t.Setenv("MYAPP_SETTING", "secret")
	fs, path := configTestMedium(t)

	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path), config.WithEnvPrefix("OTHER")))

	var setting string
	r := cfg.Get("setting", &setting)
	AssertFalse(t, r.OK)
	AssertEqual(t, "", setting)
}

func TestConfig_WithEnvPrefix_Ugly(t *T) {
	t.Setenv("MYAPP_SETTING", "secret")
	fs, path := configTestMedium(t)

	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path), config.WithEnvPrefix("MYAPP_")))

	var setting string
	r := cfg.Get("setting", &setting)
	AssertTrue(t, r.OK, r.Error())
	AssertEqual(t, "secret", setting)
}

func TestConfig_Config_Get_Good(t *T) {
	fs, path := configTestMedium(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))
	RequireTrue(t, cfg.Set("app.name", "core").OK)

	var name string
	r := cfg.Get("app.name", &name)

	AssertTrue(t, r.OK, r.Error())
	AssertEqual(t, "core", name)
}

func TestConfig_Config_Get_Bad(t *T) {
	fs, path := configTestMedium(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))

	var missing string
	r := cfg.Get("missing.key", &missing)

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "key not found")
}

func TestConfig_Config_Get_Ugly(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.yaml", "app:\n  name: core\nversion: 1\n")
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath("/config.yaml")))

	var full struct {
		App struct {
			Name string `mapstructure:"name"`
		} `mapstructure:"app"`
		Version int `mapstructure:"version"`
	}
	r := cfg.Get("", &full)

	AssertTrue(t, r.OK, r.Error())
	AssertEqual(t, "core", full.App.Name)
	AssertEqual(t, 1, full.Version)
}

func TestConfig_Config_Set_Good(t *T) {
	fs, path := configTestMedium(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))

	r := cfg.Set("dev.editor", "vim")

	AssertTrue(t, r.OK, r.Error())
	var editor string
	get := cfg.Get("dev.editor", &editor)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "vim", editor)
}

func TestConfig_Config_Set_Bad(t *T) {
	var cfg *config.Config

	r := cfg.Set("dev.editor", "vim")

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "config is nil")
}

func TestConfig_Config_Set_Ugly(t *T) {
	fs, path := configTestMedium(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))

	r := cfg.Set("feature.enabled", true)

	AssertTrue(t, r.OK, r.Error())
	var enabled bool
	get := cfg.Get("feature.enabled", &enabled)
	AssertTrue(t, get.OK, get.Error())
	AssertTrue(t, enabled)
}

func TestConfig_Config_Commit_Good(t *T) {
	fs, path := configTestMedium(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))
	RequireTrue(t, cfg.Set("dev.editor", "vim").OK)

	r := cfg.Commit()

	AssertTrue(t, r.OK, r.Error())
	content := fs.Read(path)
	RequireTrue(t, content.OK, content.Error())
	AssertContains(t, content.Value.(string), "editor: vim")
}

func TestConfig_Config_Commit_Bad(t *T) {
	fs := configTestFS(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath("/config.json")))
	RequireTrue(t, cfg.Set("key", "value").OK)

	r := cfg.Commit()

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "unsupported config file type")
}

func TestConfig_Config_Commit_Ugly(t *T) {
	fs, path := configTestMedium(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))

	r := cfg.Commit()

	AssertTrue(t, r.OK, r.Error())
	content := fs.Read(path)
	RequireTrue(t, content.OK, content.Error())
	AssertContains(t, content.Value.(string), "{}")
}

func TestConfig_Config_All_Good(t *T) {
	fs, path := configTestMedium(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))
	RequireTrue(t, cfg.Set("zulu", "last").OK)
	RequireTrue(t, cfg.Set("alpha", "first").OK)

	var keys []string
	for key := range cfg.All() {
		keys = append(keys, key)
	}

	AssertEqual(t, []string{"alpha", "zulu"}, keys)
}

func TestConfig_Config_All_Bad(t *T) {
	fs, path := configTestMedium(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))

	var keys []string
	for key := range cfg.All() {
		keys = append(keys, key)
	}

	AssertEmpty(t, keys)
}

func TestConfig_Config_All_Ugly(t *T) {
	fs, path := configTestMedium(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))
	RequireTrue(t, cfg.Set("alpha", "first").OK)
	seq := cfg.All()
	RequireTrue(t, cfg.Set("beta", "second").OK)

	var keys []string
	for key := range seq {
		keys = append(keys, key)
	}

	AssertEqual(t, []string{"alpha"}, keys)
}

func TestConfig_Config_Path_Good(t *T) {
	fs := configTestFS(t)

	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath("/custom/path/config.yaml")))

	AssertEqual(t, "/custom/path/config.yaml", cfg.Path())
}

func TestConfig_Config_Path_Bad(t *T) {
	fs := configTestFS(t)

	cfg := requireConfig(t, config.New(config.WithMedium(fs)))

	AssertContains(t, cfg.Path(), ".core/config.yaml")
}

func TestConfig_Config_Path_Ugly(t *T) {
	fs := configTestFS(t)

	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath("")))

	AssertContains(t, cfg.Path(), ".core/config.yaml")
}

func TestConfig_Config_LoadFile_Good(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, "/extra.yaml", "agent: codex\n")
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))

	r := cfg.LoadFile(fs, "/extra.yaml")

	AssertTrue(t, r.OK, r.Error())
	var agent string
	get := cfg.Get("agent", &agent)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "codex", agent)
}

func TestConfig_Config_LoadFile_Bad(t *T) {
	fs, path := configTestMedium(t)
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))

	r := cfg.LoadFile(fs, "/missing.yaml")

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "failed to read config file")
}

func TestConfig_Config_LoadFile_Ugly(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, "/.env", "TOKEN=abc\n")
	cfg := requireConfig(t, config.New(config.WithMedium(fs), config.WithPath(path)))

	r := cfg.LoadFile(fs, "/.env")

	AssertTrue(t, r.OK, r.Error())
	var token string
	get := cfg.Get("token", &token)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "abc", token)
}

func TestConfig_Load_Good(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.yaml", "app:\n  name: core\n")

	r := config.Load(fs, "/config.yaml")

	AssertTrue(t, r.OK, r.Error())
	data := r.Value.(map[string]any)
	app := data["app"].(map[string]any)
	AssertEqual(t, "core", app["name"])
}

func TestConfig_Load_Bad(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.json", `{"app":{"name":"core"}}`)

	r := config.Load(fs, "/config.json")

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "unsupported config file type")
}

func TestConfig_Load_Ugly(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config", "app:\n  name: core\n")

	r := config.Load(fs, "/config")

	AssertTrue(t, r.OK, r.Error())
	data := r.Value.(map[string]any)
	app := data["app"].(map[string]any)
	AssertEqual(t, "core", app["name"])
}

func TestConfig_Save_Good(t *T) {
	fs := configTestFS(t)

	r := config.Save(fs, "/config.yaml", map[string]any{"key": "value"})

	AssertTrue(t, r.OK, r.Error())
	content := fs.Read("/config.yaml")
	RequireTrue(t, content.OK, content.Error())
	AssertContains(t, content.Value.(string), "key: value")
}

func TestConfig_Save_Bad(t *T) {
	fs := configTestFS(t)

	r := config.Save(fs, "/config.json", map[string]any{"key": "value"})

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "unsupported config file type")
}

func TestConfig_Save_Ugly(t *T) {
	r := config.Save(refusingMedium{}, "/config.yaml", map[string]any{"key": "value"})

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "mkdir refused")
}
