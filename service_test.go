package config_test

import (
	. "dappco.re/go"
	config "dappco.re/go/config"
)

func TestService_NewConfigService_Good(t *T) {
	r := config.NewConfigService(New())

	AssertTrue(t, r.OK, r.Error())
	svc, ok := r.Value.(*config.Service)
	AssertTrue(t, ok)
	AssertNotNil(t, svc)
}

func TestService_NewConfigService_Bad(t *T) {
	r := config.NewConfigService(nil)
	RequireTrue(t, r.OK, r.Error())
	svc := r.Value.(*config.Service)

	var value string
	get := svc.Get("missing", &value)

	AssertFalse(t, get.OK)
	AssertContains(t, get.Error(), "config not loaded")
}

func TestService_NewConfigService_Ugly(t *T) {
	c := New(WithService(config.NewConfigService))

	svc, ok := ServiceFor[*config.Service](c, "config")

	AssertTrue(t, ok)
	AssertNotNil(t, svc)
}

func TestService_Service_OnStartup_Good(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, path, "app:\n  name: service\n")
	svc := &config.Service{ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: path, Medium: fs})}

	r := svc.OnStartup(Background())

	AssertTrue(t, r.OK, r.Error())
	var name string
	get := svc.Get("app.name", &name)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "service", name)
}

func TestService_Service_OnStartup_Bad(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.txt", "app.name=service")
	svc := &config.Service{ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: "/config.txt", Medium: fs})}

	r := svc.OnStartup(Background())

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "unsupported config file type")
}

func TestService_Service_OnStartup_Ugly(t *T) {
	t.Setenv("SERVICE_SETTING", "secret")
	fs, path := configTestMedium(t)
	svc := &config.Service{
		ServiceRuntime: NewServiceRuntime(nil, config.ServiceOptions{Path: path, EnvPrefix: "SERVICE", Medium: fs}),
	}

	r := svc.OnStartup(Background())

	AssertTrue(t, r.OK, r.Error())
	var setting string
	get := svc.Get("setting", &setting)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "secret", setting)
}

func TestService_Service_Get_Good(t *T) {
	fs, path := configTestMedium(t)
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})
	RequireTrue(t, svc.Set("agent", "codex").OK)

	var agent string
	r := svc.Get("agent", &agent)

	AssertTrue(t, r.OK, r.Error())
	AssertEqual(t, "codex", agent)
}

func TestService_Service_Get_Bad(t *T) {
	svc := &config.Service{}

	var agent string
	r := svc.Get("agent", &agent)

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "config not loaded")
}

func TestService_Service_Get_Ugly(t *T) {
	fs := configTestFS(t)
	writeConfigFile(t, fs, "/config.yaml", "app:\n  name: service\n")
	svc := startedService(t, config.ServiceOptions{Path: "/config.yaml", Medium: fs})

	var full struct {
		App struct {
			Name string `mapstructure:"name"`
		} `mapstructure:"app"`
	}
	r := svc.Get("", &full)

	AssertTrue(t, r.OK, r.Error())
	AssertEqual(t, "service", full.App.Name)
}

func TestService_Service_Set_Good(t *T) {
	fs, path := configTestMedium(t)
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})

	r := svc.Set("agent", "codex")

	AssertTrue(t, r.OK, r.Error())
	var agent string
	get := svc.Get("agent", &agent)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "codex", agent)
}

func TestService_Service_Set_Bad(t *T) {
	svc := &config.Service{}

	r := svc.Set("agent", "codex")

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "config not loaded")
}

func TestService_Service_Set_Ugly(t *T) {
	fs, path := configTestMedium(t)
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})

	r := svc.Set("nested.agent", "codex")

	AssertTrue(t, r.OK, r.Error())
	var agent string
	get := svc.Get("nested.agent", &agent)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "codex", agent)
}

func TestService_Service_Commit_Good(t *T) {
	fs, path := configTestMedium(t)
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})
	RequireTrue(t, svc.Set("agent", "codex").OK)

	r := svc.Commit()

	AssertTrue(t, r.OK, r.Error())
	content := fs.Read(path)
	RequireTrue(t, content.OK, content.Error())
	AssertContains(t, content.Value.(string), "agent: codex")
}

func TestService_Service_Commit_Bad(t *T) {
	svc := &config.Service{}

	r := svc.Commit()

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "config not loaded")
}

func TestService_Service_Commit_Ugly(t *T) {
	fs := configTestFS(t)
	svc := startedService(t, config.ServiceOptions{Path: "/config.json", Medium: fs})
	RequireTrue(t, svc.Set("agent", "codex").OK)

	r := svc.Commit()

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "unsupported config file type")
}

func TestService_Service_LoadFile_Good(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, "/extra.yaml", "agent: codex\n")
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})

	r := svc.LoadFile(fs, "/extra.yaml")

	AssertTrue(t, r.OK, r.Error())
	var agent string
	get := svc.Get("agent", &agent)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "codex", agent)
}

func TestService_Service_LoadFile_Bad(t *T) {
	svc := &config.Service{}
	fs, path := configTestMedium(t)

	r := svc.LoadFile(fs, path)

	AssertFalse(t, r.OK)
	AssertContains(t, r.Error(), "config not loaded")
}

func TestService_Service_LoadFile_Ugly(t *T) {
	fs, path := configTestMedium(t)
	writeConfigFile(t, fs, "/.env", "TOKEN=abc\n")
	svc := startedService(t, config.ServiceOptions{Path: path, Medium: fs})

	r := svc.LoadFile(fs, "/.env")

	AssertTrue(t, r.OK, r.Error())
	var token string
	get := svc.Get("token", &token)
	AssertTrue(t, get.OK, get.Error())
	AssertEqual(t, "abc", token)
}
