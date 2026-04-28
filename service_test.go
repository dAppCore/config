package config

import (
	"context"
	"os"
	"path/filepath"
	"runtime"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func TestService_OnStartup_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}

	result := svc.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)

	var name string
	core.AssertNoError(t, svc.Get("app.name", &name))
	core.AssertEqual(t, "svc", name)
}

func TestService_OnStartup_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/bad/config.yaml"] = "this is: [not: yaml"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/bad/config.yaml",
			Medium: m,
		}),
	}

	result := svc.OnStartup(context.Background())
	core.AssertFalse(t, result.OK)
}

func TestService_OnStartup_RegistersActions_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "dev:\n  editor: vim\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	// config.get must round-trip through the action bus.
	result := c.Action("config.get").Run(context.Background(), core.NewOptions(core.Option{Key: "key", Value: "dev.editor"}))
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "vim", result.Value)

	// config.set stores a value; config.get reads it back.
	setResult := c.Action("config.set").Run(context.Background(), core.NewOptions(
		core.Option{Key: "key", Value: "dev.shell"},
		core.Option{Key: "value", Value: "zsh"},
	))
	core.AssertTrue(t, setResult.OK)

	readResult := c.Action("config.get").Run(context.Background(), core.NewOptions(core.Option{Key: "key", Value: "dev.shell"}))
	core.AssertTrue(t, readResult.OK)
	core.AssertEqual(t, "zsh", readResult.Value)
}

func TestService_OnStartup_RegistersCommands_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	core.AssertContains(t, c.Commands(), "config/get")
	core.AssertContains(t, c.Commands(), "config/set")
	core.AssertContains(t, c.Commands(), "config/list")
}

func TestService_OnStartup_MergesProjectOverGlobal_Good(t *core.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	projectRoot := filepath.Join("/", "service-merge", "repo")
	serviceDir := filepath.Join(projectRoot, "app")

	for _, dir := range []string{
		filepath.Join(home, ".core"),
		filepath.Join(projectRoot, ".core"),
		filepath.Join(projectRoot, ".git"),
		serviceDir,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	core.AssertNoError(t, m.Write(filepath.Join(home, ".core", FileConfig), "app:\n  name: global\nservices:\n  ollama:\n    url: http://global\n"))
	core.AssertNoError(t, m.Write(filepath.Join(projectRoot, ".core", FileConfig), "app:\n  name: project\n"))

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   filepath.Join(projectRoot, ".core", FileConfig),
			Medium: m,
		}),
	}

	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	var name string
	core.AssertNoError(t, svc.Get("app.name", &name))
	core.AssertEqual(t, "project", name)

	var ollamaURL string
	core.AssertNoError(t, svc.Get("services.ollama.url", &ollamaURL))
	core.AssertEqual(t, "http://global", ollamaURL)
}

func TestService_Config_Good(t *core.T) {
	m := coreio.NewMockMedium()
	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}

	// Before OnStartup, Config() returns nil.
	core.AssertNil(t, svc.Config())

	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)
	core.AssertNotNil(t, svc.Config())
}

func TestService_Get_Bad(t *core.T) {
	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{}),
	}
	var v string
	err := svc.Get("anything", &v)
	core.AssertError(t, err)
}

func TestService_NewConfigService_Good(t *core.T) {
	// The documented usage must compile and succeed: the factory is a
	// core.WithService value and produces a retrievable *Service.
	c := core.New(core.WithService(NewConfigService))
	svc, ok := core.ServiceFor[*Service](c, "config")
	core.AssertTrue(t, ok)
	core.AssertNotNil(t, svc)
}

func TestService_NewConfigServiceWith_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/custom/config.yaml"] = "app:\n  name: custom\n"

	c := core.New(core.WithService(NewConfigServiceWith(ServiceOptions{
		Path:   "/tmp/custom/config.yaml",
		Medium: m,
	})))

	svc, ok := core.ServiceFor[*Service](c, "config")
	core.AssertTrue(t, ok)

	// OnStartup has not run yet; trigger it via the Core's service lifecycle.
	startables := c.Startables()
	core.AssertTrue(t, startables.OK)
	for _, s := range startables.Value.([]*core.Service) {
		core.AssertTrue(t, s.OnStart().OK)
	}

	var name string
	core.AssertNoError(t, svc.Get("app.name", &name))
	core.AssertEqual(t, "custom", name)
}

func TestService_NewConfigService_Bad(t *core.T) {
	// With a broken medium path (unsupported file type) OnStartup must fail
	// gracefully and return a non-OK Result rather than panicking.
	m := coreio.NewMockMedium()
	m.Files["/broken.txt"] = "ignored"
	c := core.New(core.WithService(NewConfigServiceWith(ServiceOptions{
		Path:   "/broken.txt",
		Medium: m,
	})))
	svc, ok := core.ServiceFor[*Service](c, "config")
	core.AssertTrue(t, ok)

	startables := c.Startables()
	core.AssertTrue(t, startables.OK)
	var gotFailure bool
	for _, s := range startables.Value.([]*core.Service) {
		if !s.OnStart().OK {
			gotFailure = true
		}
	}
	core.AssertTrue(t, gotFailure)
	core.AssertNil(t, svc.Config())
}

func TestService_LoadFile_RejectsUnsafePaths(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	err := svc.LoadFile(m, "../../etc/passwd")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "path traversal rejected")

	err = svc.LoadFile(m, "/etc/passwd")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "absolute config paths are not allowed")
}

func TestService_Set_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	core.AssertNoError(t, svc.Set("dev.editor", "vim"))

	var editor string
	core.AssertNoError(t, svc.Get("dev.editor", &editor))
	core.AssertEqual(t, "vim", editor)
}

func TestService_Set_Bad(t *core.T) {
	svc := &Service{}

	err := svc.Set("dev.editor", "vim")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "config not loaded")
}

func TestService_Commit_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)
	core.AssertNoError(t, svc.Set("dev.editor", "vim"))

	core.AssertNoError(t, svc.Commit())
	body, err := m.Read("/tmp/svc/config.yaml")
	core.AssertNoError(t, err)
	core.AssertContains(t, body, "editor: vim")
}

func TestService_Commit_Bad(t *core.T) {
	svc := &Service{}

	err := svc.Commit()
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "config not loaded")
}

func TestService_LoadFile_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"
	m.Files["/tmp/svc/.core/override.yaml"] = "dev:\n  shell: zsh\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	core.AssertNoError(t, svc.LoadFile(m, ".core/override.yaml"))

	var shell string
	core.AssertNoError(t, svc.Get("dev.shell", &shell))
	core.AssertEqual(t, "zsh", shell)
}

func TestService_LoadFile_Bad_NoConfig(t *core.T) {
	svc := &Service{}

	err := svc.LoadFile(coreio.NewMockMedium(), ".core/override.yaml")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "config not loaded")
}

func TestService_LoadFile_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	err := svc.LoadFile(m, filepath.Join("tmp", "svc", "config.yaml"))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "config paths must remain under .core/")
}

func TestService_LoadFile_RejectsSymlinkedCore(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	projectRoot := t.TempDir()
	externalCore := filepath.Join(t.TempDir(), "shared-core")
	core.AssertNoError(t, os.MkdirAll(externalCore, 0755))
	core.AssertNoError(t, os.WriteFile(filepath.Join(externalCore, "override.yaml"), []byte("dev:\n  shell: zsh\n"), 0600))
	core.AssertNoError(t, os.Symlink(externalCore, filepath.Join(projectRoot, ".core")))
	core.AssertNoError(t, os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte("app:\n  name: svc\n"), 0600))

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   filepath.Join(projectRoot, "config.yaml"),
			Medium: coreio.Local,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	err := svc.LoadFile(coreio.Local, ".core/override.yaml")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "symlinked .core directories are not allowed")
}

func TestService_ResolveValidatedServiceLoadPath_Good(t *core.T) {
	projectRoot := t.TempDir()
	coreDir := filepath.Join(projectRoot, ".core")
	configPath := filepath.Join(projectRoot, "config.yaml")
	overridePath := filepath.Join(coreDir, "override.yaml")

	core.AssertNoError(t, os.MkdirAll(coreDir, 0755))
	core.AssertNoError(t, os.WriteFile(configPath, []byte("app:\n  name: svc\n"), 0600))
	core.AssertNoError(t, os.WriteFile(overridePath, []byte("dev:\n  editor: vim\n"), 0600))

	resolved, err := resolveValidatedServiceLoadPath(configPath, ".core/override.yaml")
	core.AssertNoError(t, err)
	core.AssertEqual(t, overridePath, resolved)
}

func TestService_ResolveValidatedServiceLoadPath_Bad(t *core.T) {
	projectRoot := t.TempDir()
	configPath := filepath.Join(projectRoot, "config.yaml")
	core.AssertNoError(t, os.WriteFile(configPath, []byte("app:\n  name: svc\n"), 0600))

	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: "", want: "empty config path"},
		{name: "absolute", path: "/etc/passwd", want: "absolute config paths are not allowed"},
		{name: "traversal", path: "../escape.yaml", want: "path traversal rejected"},
		{name: "outside-core", path: "config.yaml", want: "config paths must remain under .core/"},
		{name: "nested-traversal", path: ".core/../escape.yaml", want: "config paths must remain under .core/"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *core.T) {
			resolved, err := resolveValidatedServiceLoadPath(configPath, tc.path)
			core.AssertEmpty(t, resolved)
			core.AssertError(t, err)
			core.AssertContains(t, err.Error(), tc.want)
		})
	}
}

func TestService_ResolveValidatedServiceLoadPath_Ugly(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	projectRoot := t.TempDir()
	externalCore := filepath.Join(t.TempDir(), "shared-core")
	configPath := filepath.Join(projectRoot, "config.yaml")

	core.AssertNoError(t, os.MkdirAll(externalCore, 0755))
	core.AssertNoError(t, os.WriteFile(configPath, []byte("app:\n  name: svc\n"), 0600))
	core.AssertNoError(t, os.Symlink(externalCore, filepath.Join(projectRoot, ".core")))

	resolved, err := resolveValidatedServiceLoadPath(configPath, ".core/override.yaml")
	core.AssertEmpty(t, resolved)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "symlinked .core directories are not allowed")
}

func TestService_ResolveServiceLoadPath_Good(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	projectRoot := t.TempDir()
	coreDir := filepath.Join(projectRoot, ".core")
	realFile := filepath.Join(projectRoot, "override.yaml")
	symlinkFile := filepath.Join(coreDir, "override.yaml")

	core.AssertNoError(t, os.MkdirAll(coreDir, 0755))
	core.AssertNoError(t, os.WriteFile(realFile, []byte("dev:\n  shell: zsh\n"), 0600))
	core.AssertNoError(t, os.Symlink(realFile, symlinkFile))

	absCorePath, err := filepath.Abs(coreDir)
	core.AssertNoError(t, err)
	absCandidate, err := filepath.Abs(symlinkFile)
	core.AssertNoError(t, err)

	resolvedCandidate, resolvedCore, err := resolveServiceLoadPath(symlinkFile, absCorePath, absCandidate)
	core.AssertNoError(t, err)
	realCandidate, err := filepath.EvalSymlinks(realFile)
	core.AssertNoError(t, err)
	realCore, err := filepath.EvalSymlinks(coreDir)
	core.AssertNoError(t, err)
	core.AssertEqual(t, realCandidate, resolvedCandidate)
	core.AssertEqual(t, realCore, resolvedCore)
}

func TestService_ResolveServiceLoadPath_Bad(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	projectRoot := t.TempDir()
	externalCore := filepath.Join(t.TempDir(), "shared-core")
	candidatePath := filepath.Join(projectRoot, ".core", "override.yaml")

	core.AssertNoError(t, os.MkdirAll(externalCore, 0755))
	core.AssertNoError(t, os.Symlink(externalCore, filepath.Join(projectRoot, ".core")))

	absCorePath, err := filepath.Abs(filepath.Join(projectRoot, ".core"))
	core.AssertNoError(t, err)
	absCandidate, err := filepath.Abs(candidatePath)
	core.AssertNoError(t, err)

	resolvedCandidate, resolvedCore, err := resolveServiceLoadPath(candidatePath, absCorePath, absCandidate)
	core.AssertEmpty(t, resolvedCandidate)
	core.AssertEmpty(t, resolvedCore)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "symlinked .core directories are not allowed")
}

func TestService_OnShutdown_StopsWatcher_Good(t *core.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	core.AssertNoError(t, coreio.Local.Write(path, "app:\n  name: svc\n"))

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   path,
			Medium: coreio.Local,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)
	core.AssertNoError(t, svc.Config().Watch())
	core.AssertNotNil(t, svc.Config().watcher)

	result := svc.OnShutdown(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, svc.Config().watcher)
}

func TestService_RegistersActionsAndCommands_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"
	m.Files["/tmp/svc/.core/loaded.yaml"] = "dev:\n  editor: nano\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	runAction := func(name string, opts core.Options) core.Result {
		return c.Action(name).Run(context.Background(), opts)
	}
	runCommand := func(name string, opts core.Options) core.Result {
		r := c.Command(name)
		if !r.OK {
			core.AssertTrue(t, r.OK)
			return r
		}
		return r.Value.(*core.Command).Run(opts)
	}

	core.AssertTrue(t, runAction("config.get", core.NewOptions(core.Option{Key: "key", Value: "app.name"})).OK)
	core.AssertTrue(t, runAction("config.set", core.NewOptions(
		core.Option{Key: "key", Value: "dev.shell"},
		core.Option{Key: "value", Value: "zsh"},
	)).OK)
	core.AssertTrue(t, runAction("config.commit", core.NewOptions()).OK)
	core.AssertTrue(t, runAction("config.load", core.NewOptions(core.Option{Key: "path", Value: ".core/loaded.yaml"})).OK)

	all := runAction("config.all", core.NewOptions())
	core.AssertTrue(t, all.OK)
	core.AssertContains(t, all.Value.(map[string]any), "app.name")

	path := runAction("config.path", core.NewOptions())
	core.AssertTrue(t, path.OK)
	core.AssertEqual(t, "/tmp/svc/config.yaml", path.Value)

	core.AssertTrue(t, runCommand("config/get", core.NewOptions(core.Option{Key: "key", Value: "app.name"})).OK)
	core.AssertTrue(t, runCommand("config/set", core.NewOptions(
		core.Option{Key: "key", Value: "dev.theme"},
		core.Option{Key: "value", Value: "dark"},
	)).OK)
	core.AssertTrue(t, runCommand("config/commit", core.NewOptions()).OK)
	core.AssertTrue(t, runCommand("config/load", core.NewOptions(core.Option{Key: "path", Value: ".core/loaded.yaml"})).OK)
	core.AssertTrue(t, runCommand("config/list", core.NewOptions()).OK)
	core.AssertTrue(t, runCommand("config/path", core.NewOptions()).OK)
}

func TestService_ReadCommands_RequireEntitlement(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	c.SetEntitlementChecker(func(action string, qty int, _ context.Context) core.Entitlement {
		_ = qty
		switch action {
		case "config/get", "config/list", "config/path":
			return core.Entitlement{Allowed: false, Reason: "denied"}
		default:
			return core.Entitlement{Allowed: true, Unlimited: true}
		}
	})

	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	for _, name := range []string{"config/get", "config/list", "config/path"} {
		cmdResult := c.Command(name)
		if !cmdResult.OK {
			core.AssertTrue(t, cmdResult.OK, name)
			continue
		}
		res := cmdResult.Value.(*core.Command).Run(core.NewOptions())
		core.AssertFalse(t, res.OK, name)
		core.AssertContains(t, res.Value.(error).Error(), "not entitled")
	}
}

func TestService_ReadActions_RequireEntitlement_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	c.SetEntitlementChecker(func(action string, qty int, _ context.Context) core.Entitlement {
		_ = qty
		switch action {
		case "config.get", "config.all", "config.path":
			return core.Entitlement{Allowed: false, Reason: "denied"}
		default:
			return core.Entitlement{Allowed: true, Unlimited: true}
		}
	})

	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	actions := map[string]core.Options{
		"config.get":  core.NewOptions(core.Option{Key: "key", Value: "app.name"}),
		"config.all":  core.NewOptions(),
		"config.path": core.NewOptions(),
	}

	for name, opts := range actions {
		t.Run(name, func(t *core.T) {
			res := c.Action(name).Run(context.Background(), opts)
			core.AssertFalse(t, res.OK)
			core.AssertContains(t, res.Value.(error).Error(), "not entitled")
		})
	}
}

func axServiceFixture(t *core.T) (*Service, *coreio.MockMedium, string) {
	t.Helper()
	m := coreio.NewMockMedium()
	path := "/ax7/service/config.yaml"
	m.Files[path] = "app:\n  name: svc\n"
	c := core.New()
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{Path: path, Medium: m})}
	core.RequireTrue(t, svc.OnStartup(context.Background()).OK)
	return svc, m, path
}

func TestService_NewConfigService_Ugly(t *core.T) {
	result := NewConfigService(nil)
	core.AssertTrue(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestService_NewConfigServiceWith_Bad(t *core.T) {
	factory := NewConfigServiceWith(ServiceOptions{})
	result := factory(core.New())
	core.AssertTrue(t, result.OK)
	core.AssertNotNil(t, result.Value)
}

func TestService_NewConfigServiceWith_Ugly(t *core.T) {
	factory := NewConfigServiceWith(ServiceOptions{Path: "/ax7/config.yaml", EnvPrefix: "AX7"})
	result := factory(nil)
	svc := result.Value.(*Service)
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "/ax7/config.yaml", svc.Options().Path)
}

func TestService_Service_OnStartup_Good(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	var got string
	err := svc.Get("app.name", &got)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "svc", got)
}

func TestService_Service_OnStartup_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/ax7/bad.yaml"] = "app: [broken"
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{Path: "/ax7/bad.yaml", Medium: m})}
	result := svc.OnStartup(context.Background())
	core.AssertFalse(t, result.OK)
}

func TestService_Service_OnStartup_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{Path: "/ax7/empty.yaml", Medium: m})}
	result := svc.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNotNil(t, svc.Config())
}

func TestService_Service_OnShutdown_Good(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	result := svc.OnShutdown(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNotNil(t, svc.Config())
}

func TestService_Service_OnShutdown_Bad(t *core.T) {
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{})}
	result := svc.OnShutdown(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, svc.Config())
}

func TestService_Service_OnShutdown_Ugly(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	first := svc.OnShutdown(context.Background())
	second := svc.OnShutdown(context.Background())
	core.AssertTrue(t, first.OK)
	core.AssertTrue(t, second.OK)
}

func TestService_Service_Get_Good(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	var got string
	err := svc.Get("app.name", &got)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "svc", got)
}

func TestService_Service_Get_Bad(t *core.T) {
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{})}
	err := svc.Get("missing", new(string))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_Service_Get_Ugly(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	var got map[string]any
	err := svc.Get("", &got)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "svc", got["app"].(map[string]any)["name"])
}

func TestService_Service_Set_Good(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	err := svc.Set("dev.editor", "vim")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "vim", configValues(svc.Config())["dev.editor"])
}

func TestService_Service_Set_Bad(t *core.T) {
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{})}
	err := svc.Set("dev.editor", "vim")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_Service_Set_Ugly(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	err := svc.Set("", "root")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "root", svc.Config().file.Get(""))
}

func TestService_Service_Commit_Good(t *core.T) {
	svc, m, path := axServiceFixture(t)
	core.RequireNoError(t, svc.Set("dev.editor", "vim"))
	err := svc.Commit()
	core.AssertNoError(t, err)
	core.AssertTrue(t, m.Exists(path))
}

func TestService_Service_Commit_Bad(t *core.T) {
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{})}
	err := svc.Commit()
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_Service_Commit_Ugly(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	svc.Config().path = "/ax7/config.json"
	err := svc.Commit()
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsupported config file type")
}

func TestService_Service_LoadFile_Good(t *core.T) {
	svc, m, _ := axServiceFixture(t)
	core.RequireNoError(t, m.Write("/ax7/service/.core/override.yaml", "dev:\n  shell: zsh\n"))
	err := svc.LoadFile(m, ".core/override.yaml")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "zsh", configValues(svc.Config())["dev.shell"])
}

func TestService_Service_LoadFile_Bad(t *core.T) {
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{})}
	err := svc.LoadFile(coreio.NewMockMedium(), ".core/override.yaml")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_Service_LoadFile_Ugly(t *core.T) {
	svc, m, _ := axServiceFixture(t)
	err := svc.LoadFile(m, "../escape.yaml")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "path traversal")
}

func TestService_Service_Config_Good(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	got := svc.Config()
	core.AssertNotNil(t, got)
}

func TestService_Service_Config_Bad(t *core.T) {
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{})}
	got := svc.Config()
	core.AssertNil(t, got)
}

func TestService_Service_Config_Ugly(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	replacement, err := New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/replacement.yaml"))
	core.RequireNoError(t, err)
	svc.config = replacement
	core.AssertSame(t, replacement, svc.Config())
}
