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

	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))

	AssertNoError(t, err)
	var name string
	AssertNoError(t, cfg.Get("app.name", &name))
	AssertEqual(t, "core", name)
}

func TestConfig_New_Bad(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.txt", "app.name=core")

	cfg, err := config.New(config.WithMedium(fs), config.WithPath("/config.txt"))

	AssertNil(t, cfg)
	AssertError(t, err)
	AssertContains(t, err.Error(), "unsupported config file type")
}

func TestConfig_New_Ugly(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/.env", "FOO=bar\n")

	cfg, err := config.New(config.WithMedium(fs), config.WithPath("/.env"))

	AssertNoError(t, err)
	var foo string
	AssertNoError(t, cfg.Get("foo", &foo))
	AssertEqual(t, "bar", foo)
}

func TestConfig_WithMedium_Good(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, path, "agent: codex\n")

	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))

	AssertNoError(t, err)
	var agent string
	AssertNoError(t, cfg.Get("agent", &agent))
	AssertEqual(t, "codex", agent)
}

func TestConfig_WithMedium_Bad(t *T) {
	cfg, err := config.New(config.WithMedium(nil), config.WithPath("/missing.yaml"))

	AssertNoError(t, err)
	AssertNotNil(t, cfg)
	AssertEqual(t, "/missing.yaml", cfg.Path())
}

func TestConfig_WithMedium_Ugly(t *T) {
	cfg, err := config.New(config.WithMedium(refusingMedium{}), config.WithPath("/config.yaml"))

	AssertNil(t, cfg)
	AssertError(t, err)
	AssertContains(t, err.Error(), "read refused")
}

func TestConfig_WithPath_Good(t *T) {
	fs := configTestFS(t)

	cfg, err := config.New(config.WithMedium(fs), config.WithPath("/custom/config.yaml"))

	AssertNoError(t, err)
	AssertEqual(t, "/custom/config.yaml", cfg.Path())
}

func TestConfig_WithPath_Bad(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/custom/config.json", `{"app":{"name":"core"}}`)

	cfg, err := config.New(config.WithMedium(fs), config.WithPath("/custom/config.json"))

	AssertNoError(t, err)
	AssertEqual(t, "/custom/config.json", cfg.Path())
}

func TestConfig_WithPath_Ugly(t *T) {
	fs := configTestFS(t)

	cfg, err := config.New(config.WithMedium(fs), config.WithPath(""))

	AssertNoError(t, err)
	AssertContains(t, cfg.Path(), ".core/config.yaml")
}

func TestConfig_WithEnvPrefix_Good(t *T) {
	t.Setenv("MYAPP_SETTING", "secret")
	fs, path := configTestMedium(t)

	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path), config.WithEnvPrefix("MYAPP"))

	AssertNoError(t, err)
	var setting string
	AssertNoError(t, cfg.Get("setting", &setting))
	AssertEqual(t, "secret", setting)
}

func TestConfig_WithEnvPrefix_Bad(t *T) {
	t.Setenv("MYAPP_SETTING", "secret")
	fs, path := configTestMedium(t)

	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path), config.WithEnvPrefix("OTHER"))

	AssertNoError(t, err)
	var setting string
	AssertError(t, cfg.Get("setting", &setting))
	AssertEqual(t, "", setting)
}

func TestConfig_WithEnvPrefix_Ugly(t *T) {
	t.Setenv("MYAPP_SETTING", "secret")
	fs, path := configTestMedium(t)

	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path), config.WithEnvPrefix("MYAPP_"))

	AssertNoError(t, err)
	var setting string
	AssertNoError(t, cfg.Get("setting", &setting))
	AssertEqual(t, "secret", setting)
}

func TestConfig_Config_Get_Good(t *T) {
	fs, path := configTestMedium(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)
	RequireNoError(t, cfg.Set("app.name", "core"))

	var name string
	err = cfg.Get("app.name", &name)

	AssertNoError(t, err)
	AssertEqual(t, "core", name)
}

func TestConfig_Config_Get_Bad(t *T) {
	fs, path := configTestMedium(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)

	var missing string
	err = cfg.Get("missing.key", &missing)

	AssertError(t, err)
	AssertContains(t, err.Error(), "key not found")
}

func TestConfig_Config_Get_Ugly(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.yaml", "app:\n  name: core\nversion: 1\n")
	cfg, err := config.New(config.WithMedium(fs), config.WithPath("/config.yaml"))
	RequireNoError(t, err)

	var full struct {
		App struct {
			Name string `mapstructure:"name"`
		} `mapstructure:"app"`
		Version int `mapstructure:"version"`
	}
	err = cfg.Get("", &full)

	AssertNoError(t, err)
	AssertEqual(t, "core", full.App.Name)
	AssertEqual(t, 1, full.Version)
}

func TestConfig_Config_Set_Good(t *T) {
	fs, path := configTestMedium(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)

	err = cfg.Set("dev.editor", "vim")

	AssertNoError(t, err)
	var editor string
	AssertNoError(t, cfg.Get("dev.editor", &editor))
	AssertEqual(t, "vim", editor)
}

func TestConfig_Config_Set_Bad(t *T) {
	var cfg *config.Config

	AssertPanics(t, func() {
		_ = cfg.Set("dev.editor", "vim")
	})
}

func TestConfig_Config_Set_Ugly(t *T) {
	fs, path := configTestMedium(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)

	err = cfg.Set("", "root-value")

	AssertNoError(t, err)
	var full map[string]any
	AssertNoError(t, cfg.Get("", &full))
	AssertEqual(t, "root-value", full[""])
}

func TestConfig_Config_Commit_Good(t *T) {
	fs, path := configTestMedium(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)
	RequireNoError(t, cfg.Set("dev.editor", "vim"))

	err = cfg.Commit()

	AssertNoError(t, err)
	content := fs.Read(path)
	RequireTrue(t, content.OK, content.Error())
	AssertContains(t, content.Value.(string), "editor: vim")
}

func TestConfig_Config_Commit_Bad(t *T) {
	fs := configTestFS(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath("/config.json"))
	RequireNoError(t, err)
	RequireNoError(t, cfg.Set("key", "value"))

	err = cfg.Commit()

	AssertError(t, err)
	AssertContains(t, err.Error(), "unsupported config file type")
}

func TestConfig_Config_Commit_Ugly(t *T) {
	fs, path := configTestMedium(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)

	err = cfg.Commit()

	AssertNoError(t, err)
	content := fs.Read(path)
	RequireTrue(t, content.OK, content.Error())
	AssertContains(t, content.Value.(string), "{}")
}

func TestConfig_Config_All_Good(t *T) {
	fs, path := configTestMedium(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)
	RequireNoError(t, cfg.Set("zulu", "last"))
	RequireNoError(t, cfg.Set("alpha", "first"))

	var keys []string
	for key := range cfg.All() {
		keys = append(keys, key)
	}

	AssertEqual(t, []string{"alpha", "zulu"}, keys)
}

func TestConfig_Config_All_Bad(t *T) {
	fs, path := configTestMedium(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)

	var keys []string
	for key := range cfg.All() {
		keys = append(keys, key)
	}

	AssertEmpty(t, keys)
}

func TestConfig_Config_All_Ugly(t *T) {
	fs, path := configTestMedium(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)
	RequireNoError(t, cfg.Set("alpha", "first"))
	seq := cfg.All()
	RequireNoError(t, cfg.Set("beta", "second"))

	var keys []string
	for key := range seq {
		keys = append(keys, key)
	}

	AssertEqual(t, []string{"alpha"}, keys)
}

func TestConfig_Config_Path_Good(t *T) {
	fs := configTestFS(t)

	cfg, err := config.New(config.WithMedium(fs), config.WithPath("/custom/path/config.yaml"))

	AssertNoError(t, err)
	AssertEqual(t, "/custom/path/config.yaml", cfg.Path())
}

func TestConfig_Config_Path_Bad(t *T) {
	fs := configTestFS(t)

	cfg, err := config.New(config.WithMedium(fs))

	AssertNoError(t, err)
	AssertContains(t, cfg.Path(), ".core/config.yaml")
}

func TestConfig_Config_Path_Ugly(t *T) {
	fs := configTestFS(t)

	cfg, err := config.New(config.WithMedium(fs), config.WithPath(""))

	AssertNoError(t, err)
	AssertContains(t, cfg.Path(), ".core/config.yaml")
}

func TestConfig_Config_LoadFile_Good(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, "/extra.yaml", "agent: codex\n")
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)

	err = cfg.LoadFile(fs, "/extra.yaml")

	AssertNoError(t, err)
	var agent string
	AssertNoError(t, cfg.Get("agent", &agent))
	AssertEqual(t, "codex", agent)
}

func TestConfig_Config_LoadFile_Bad(t *T) {
	fs, path := configTestMedium(t)
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)

	err = cfg.LoadFile(fs, "/missing.yaml")

	AssertError(t, err)
	AssertContains(t, err.Error(), "failed to read config file")
}

func TestConfig_Config_LoadFile_Ugly(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, "/.env", "TOKEN=abc\n")
	cfg, err := config.New(config.WithMedium(fs), config.WithPath(path))
	RequireNoError(t, err)

	err = cfg.LoadFile(fs, "/.env")

	AssertNoError(t, err)
	var token string
	AssertNoError(t, cfg.Get("token", &token))
	AssertEqual(t, "abc", token)
}

func TestConfig_Load_Good(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.yaml", "app:\n  name: core\n")

	data, err := config.Load(fs, "/config.yaml")

	AssertNoError(t, err)
	app := data["app"].(map[string]any)
	AssertEqual(t, "core", app["name"])
}

func TestConfig_Load_Bad(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.json", `{"app":{"name":"core"}}`)

	data, err := config.Load(fs, "/config.json")

	AssertNil(t, data)
	AssertError(t, err)
	AssertContains(t, err.Error(), "unsupported config file type")
}

func TestConfig_Load_Ugly(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config", "app:\n  name: core\n")

	data, err := config.Load(fs, "/config")

	AssertNoError(t, err)
	app := data["app"].(map[string]any)
	AssertEqual(t, "core", app["name"])
}

func TestConfig_Save_Good(t *T) {
	fs := configTestFS(t)

	err := config.Save(fs, "/config.yaml", map[string]any{"key": "value"})

	AssertNoError(t, err)
	content := fs.Read("/config.yaml")
	RequireTrue(t, content.OK, content.Error())
	AssertContains(t, content.Value.(string), "key: value")
}

func TestConfig_Save_Bad(t *T) {
	fs := configTestFS(t)

	err := config.Save(fs, "/config.json", map[string]any{"key": "value"})

	AssertError(t, err)
	AssertContains(t, err.Error(), "unsupported config file type")
}

func TestConfig_Save_Ugly(t *T) {
	err := config.Save(refusingMedium{}, "/config.yaml", map[string]any{"key": "value"})

	AssertError(t, err)
	AssertContains(t, err.Error(), "mkdir refused")
}

func TestConfig_Env_Good(t *T) {
	t.Setenv("AX_CONFIG_FOO_BAR", "baz")
	t.Setenv("AX_CONFIG_ALPHA", "first")

	var keys []string
	var values []any
	for key, value := range config.Env("AX_CONFIG_") {
		keys = append(keys, key)
		values = append(values, value)
	}

	AssertEqual(t, []string{"alpha", "foo.bar"}, keys)
	AssertEqual(t, []any{"first", "baz"}, values)
}

func TestConfig_Env_Bad(t *T) {
	t.Setenv("AX_CONFIG_FOO", "bar")

	var keys []string
	for key := range config.Env("OTHER_CONFIG_") {
		keys = append(keys, key)
	}

	AssertEmpty(t, keys)
}

func TestConfig_Env_Ugly(t *T) {
	t.Setenv("AX_CONFIG_FOO_BAR", "baz")

	var keys []string
	for key := range config.Env("AX_CONFIG") {
		keys = append(keys, key)
	}

	AssertEqual(t, []string{"foo.bar"}, keys)
}

func TestConfig_LoadEnv_Good(t *T) {
	t.Setenv("AX_CONFIG_FOO_BAR", "baz")

	data := config.LoadEnv("AX_CONFIG_")

	AssertLen(t, data, 1)
	AssertEqual(t, "baz", data["foo.bar"])
}

func TestConfig_LoadEnv_Bad(t *T) {
	t.Setenv("AX_CONFIG_FOO_BAR", "baz")

	data := config.LoadEnv("OTHER_CONFIG_")

	AssertEmpty(t, data)
}

func TestConfig_LoadEnv_Ugly(t *T) {
	t.Setenv("AX_CONFIG_FOO_BAR", "baz")

	data := config.LoadEnv("AX_CONFIG")

	AssertLen(t, data, 1)
	AssertEqual(t, "baz", data["foo.bar"])
}

func TestConfig_Service_OnStartup_Good(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, path, "app:\n  name: service\n")
	svc := &config.Service{ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: path, Medium: fs})}

	r := svc.OnStartup(Background())

	AssertTrue(t, r.OK, r.Error())
	var name string
	AssertNoError(t, svc.Get("app.name", &name))
	AssertEqual(t, "service", name)
}

func TestConfig_Service_OnStartup_Bad(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.txt", "app.name=service")
	svc := &config.Service{ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: "/config.txt", Medium: fs})}

	r := svc.OnStartup(Background())

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "unsupported config file type")
}

func TestConfig_Service_OnStartup_Ugly(t *T) {
	t.Setenv("SERVICE_SETTING", "secret")
	fs, path := configTestMedium(t)
	svc := &config.Service{
		ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: path, EnvPrefix: "SERVICE", Medium: fs}),
	}

	r := svc.OnStartup(Background())

	AssertTrue(t, r.OK, r.Error())
	var setting string
	AssertNoError(t, svc.Get("setting", &setting))
	AssertEqual(t, "secret", setting)
}

func TestConfig_Service_Get_Good(t *T) {
	fs, path := configTestMedium(t)
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})
	RequireNoError(t, svc.Set("agent", "codex"))

	var agent string
	err := svc.Get("agent", &agent)

	AssertNoError(t, err)
	AssertEqual(t, "codex", agent)
}

func TestConfig_Service_Get_Bad(t *T) {
	svc := &config.Service{}

	var agent string
	err := svc.Get("agent", &agent)

	AssertError(t, err)
	AssertContains(t, err.Error(), "config not loaded")
}

func TestConfig_Service_Get_Ugly(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.yaml", "app:\n  name: service\n")
	svc := startedService(t, config.ServiceOptions{Path: "/config.yaml", Medium: fs})

	var full struct {
		App struct {
			Name string `mapstructure:"name"`
		} `mapstructure:"app"`
	}
	err := svc.Get("", &full)

	AssertNoError(t, err)
	AssertEqual(t, "service", full.App.Name)
}

func TestConfig_Service_Set_Good(t *T) {
	fs, path := configTestMedium(t)
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})

	err := svc.Set("agent", "codex")

	AssertNoError(t, err)
	var agent string
	AssertNoError(t, svc.Get("agent", &agent))
	AssertEqual(t, "codex", agent)
}

func TestConfig_Service_Set_Bad(t *T) {
	svc := &config.Service{}

	err := svc.Set("agent", "codex")

	AssertError(t, err)
	AssertContains(t, err.Error(), "config not loaded")
}

func TestConfig_Service_Set_Ugly(t *T) {
	fs, path := configTestMedium(t)
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})

	err := svc.Set("nested.agent", "codex")

	AssertNoError(t, err)
	var agent string
	AssertNoError(t, svc.Get("nested.agent", &agent))
	AssertEqual(t, "codex", agent)
}

func TestConfig_Service_Commit_Good(t *T) {
	fs, path := configTestMedium(t)
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})
	RequireNoError(t, svc.Set("agent", "codex"))

	err := svc.Commit()

	AssertNoError(t, err)
	content := fs.Read(path)
	RequireTrue(t, content.OK, content.Error())
	AssertContains(t, content.Value.(string), "agent: codex")
}

func TestConfig_Service_Commit_Bad(t *T) {
	svc := &config.Service{}

	err := svc.Commit()

	AssertError(t, err)
	AssertContains(t, err.Error(), "config not loaded")
}

func TestConfig_Service_Commit_Ugly(t *T) {
	fs := configTestFS(t)
	svc := startedService(t, config.ServiceOptions{Path: "/config.json", Medium: fs})
	RequireNoError(t, svc.Set("agent", "codex"))

	err := svc.Commit()

	AssertError(t, err)
	AssertContains(t, err.Error(), "unsupported config file type")
}

func TestConfig_Service_LoadFile_Good(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, "/extra.yaml", "agent: codex\n")
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})

	err := svc.LoadFile(fs, "/extra.yaml")

	AssertNoError(t, err)
	var agent string
	AssertNoError(t, svc.Get("agent", &agent))
	AssertEqual(t, "codex", agent)
}

func TestConfig_Service_LoadFile_Bad(t *T) {
	svc := &config.Service{}
	fs, path := configTestMedium(t)

	err := svc.LoadFile(fs, path)

	AssertError(t, err)
	AssertContains(t, err.Error(), "config not loaded")
}

func TestConfig_Service_LoadFile_Ugly(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, "/.env", "TOKEN=abc\n")
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})

	err := svc.LoadFile(fs, "/.env")

	AssertNoError(t, err)
	var token string
	AssertNoError(t, svc.Get("token", &token))
	AssertEqual(t, "abc", token)
}

func TestConfig_NewConfigService_Good(t *T) {
	r := config.NewConfigService(New())

	AssertTrue(t, r.OK, r.Error())
	svc, ok := r.Value.(*config.Service)
	AssertTrue(t, ok)
	AssertNotNil(t, svc)
}

func TestConfig_NewConfigService_Bad(t *T) {
	r := config.NewConfigService(nil)
	RequireTrue(t, r.OK, r.Error())
	svc := r.Value.(*config.Service)

	var value string
	err := svc.Get("missing", &value)

	AssertError(t, err)
	AssertContains(t, err.Error(), "config not loaded")
}

func TestConfig_NewConfigService_Ugly(t *T) {
	c := New(WithService(config.NewConfigService))

	svc, ok := ServiceFor[*config.Service](c, "config")

	AssertTrue(t, ok)
	AssertNotNil(t, svc)
}
