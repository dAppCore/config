package config

import (
	"context"
	"runtime"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

const (
	serviceTestConfigPath                = "/tmp/svc/" + FileConfig
	serviceTestConfigBody                = "app:\n  name: svc\n"
	serviceTestBadConfigPath             = "/bad/" + FileConfig
	serviceTestCustomConfigPath          = "/tmp/custom/" + FileConfig
	serviceTestDevEditorKey              = "dev.editor"
	serviceTestAppNameKey                = "app.name"
	serviceTestDevShellKey               = "dev.shell"
	serviceTestShellYAML                 = "dev:\n  shell: zsh\n"
	serviceTestConfigGetAction           = "config.get"
	serviceTestConfigSetAction           = "config.set"
	serviceTestConfigCommitAction        = "config.commit"
	serviceTestConfigLoadAction          = "config.load"
	serviceTestConfigGetCommand          = "config/get"
	serviceTestConfigSetCommand          = "config/set"
	serviceTestConfigCommitCommand       = "config/commit"
	serviceTestConfigLoadCommand         = "config/load"
	serviceTestConfigListCommand         = "config/list"
	serviceTestConfigAllAction           = "config.all"
	serviceTestConfigPathAction          = "config.path"
	serviceTestConfigPathCommand         = "config/path"
	serviceTestOverridePath              = ".core/override.yaml"
	serviceTestConfigPathsUnderCore      = "config paths must remain under .core/"
	serviceTestSharedCoreDir             = "shared-core"
	serviceTestOverrideFilename          = "override.yaml"
	serviceTestSymlinkedCoreDirsRejected = "symlinked .core directories are not allowed"
	serviceTestWindowsSymlinkSkipMessage = "symlink test is not portable on Windows in this environment"
)

func TestService_OnStartup_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestConfigPath] = serviceTestConfigBody

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
			Medium: m,
		}),
	}

	result := svc.OnStartup(context.Background())
	core.AssertTrue(t, result.OK)

	var name string
	core.AssertNoError(t, resultError(svc.Get(serviceTestAppNameKey, &name)))
	core.AssertEqual(t, "svc", name)
}

func TestService_OnStartup_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestBadConfigPath] = "this is: [not: yaml"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestBadConfigPath,
			Medium: m,
		}),
	}

	result := svc.OnStartup(context.Background())
	core.AssertFalse(t, result.OK)
}

func TestService_OnStartup_RegistersActions_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestConfigPath] = "dev:\n  editor: vim\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	// config.get must round-trip through the action bus.
	result := c.Action(serviceTestConfigGetAction).Run(context.Background(), core.NewOptions(core.Option{Key: "key", Value: serviceTestDevEditorKey}))
	core.AssertTrue(t, result.OK)
	core.AssertEqual(t, "vim", result.Value)

	// config.set stores a value; config.get reads it back.
	setResult := c.Action(serviceTestConfigSetAction).Run(context.Background(), core.NewOptions(
		core.Option{Key: "key", Value: serviceTestDevShellKey},
		core.Option{Key: "value", Value: "zsh"},
	))
	core.AssertTrue(t, setResult.OK)

	readResult := c.Action(serviceTestConfigGetAction).Run(context.Background(), core.NewOptions(core.Option{Key: "key", Value: serviceTestDevShellKey}))
	core.AssertTrue(t, readResult.OK)
	core.AssertEqual(t, "zsh", readResult.Value)
}

func TestService_OnStartup_RegistersCommands_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestConfigPath] = serviceTestConfigBody

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	core.AssertContains(t, c.Commands(), serviceTestConfigGetCommand)
	core.AssertContains(t, c.Commands(), serviceTestConfigSetCommand)
	core.AssertContains(t, c.Commands(), serviceTestConfigListCommand)
}

func TestService_OnStartup_MergesProjectOverGlobal_Good(t *core.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	projectRoot := core.PathJoin("/", "service-merge", "repo")
	serviceDir := core.PathJoin(projectRoot, "app")

	for _, dir := range []string{
		core.PathJoin(home, ".core"),
		core.PathJoin(projectRoot, ".core"),
		core.PathJoin(projectRoot, ".git"),
		serviceDir,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	core.AssertNoError(t, m.Write(core.PathJoin(home, ".core", FileConfig), "app:\n  name: global\nservices:\n  ollama:\n    url: http://global\n"))
	core.AssertNoError(t, m.Write(core.PathJoin(projectRoot, ".core", FileConfig), "app:\n  name: project\n"))

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   core.PathJoin(projectRoot, ".core", FileConfig),
			Medium: m,
		}),
	}

	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	var name string
	core.AssertNoError(t, resultError(svc.Get(serviceTestAppNameKey, &name)))
	core.AssertEqual(t, "project", name)

	var ollamaURL string
	core.AssertNoError(t, resultError(svc.Get("services.ollama.url", &ollamaURL)))
	core.AssertEqual(t, "http://global", ollamaURL)
}

func TestService_Config_Good(t *core.T) {
	m := coreio.NewMockMedium()
	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
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
	err := resultError(svc.Get("anything", &v))
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
	m.Files[serviceTestCustomConfigPath] = "app:\n  name: custom\n"

	c := core.New(core.WithService(NewConfigServiceWith(ServiceOptions{
		Path:   serviceTestCustomConfigPath,
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
	core.AssertNoError(t, resultError(svc.Get(serviceTestAppNameKey, &name)))
	core.AssertEqual(t, "custom", name)
}

func TestService_NewConfigService_Bad(t *core.T) {
	// With a broken medium path (unsupported file type) OnStartup must fail
	// gracefully and return a non-OK Result rather than panicking.
	m := coreio.NewMockMedium()
	m.Files["/broken.txt"] = "ignored"
	direct := NewConfigService(core.New())
	core.AssertTrue(t, direct.OK)

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
	m.Files[serviceTestConfigPath] = serviceTestConfigBody

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	err := resultError(svc.LoadFile(m, "../../etc/passwd"))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "path traversal rejected")

	err = resultError(svc.LoadFile(m, "/etc/passwd"))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "absolute config paths are not allowed")
}

func TestService_Set_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestConfigPath] = serviceTestConfigBody

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	core.AssertNoError(t, resultError(svc.Set(serviceTestDevEditorKey, "vim")))

	var editor string
	core.AssertNoError(t, resultError(svc.Get(serviceTestDevEditorKey, &editor)))
	core.AssertEqual(t, "vim", editor)
}

func TestService_Set_Bad(t *core.T) {
	svc := &Service{}

	err := resultError(svc.Set(serviceTestDevEditorKey, "vim"))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_Commit_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestConfigPath] = serviceTestConfigBody

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)
	core.AssertNoError(t, resultError(svc.Set(serviceTestDevEditorKey, "vim")))

	core.AssertNoError(t, resultError(svc.Commit()))
	body, err := m.Read(serviceTestConfigPath)
	core.AssertNoError(t, err)
	core.AssertContains(t, body, "editor: vim")
}

func TestService_Commit_Bad(t *core.T) {
	svc := &Service{}

	err := resultError(svc.Commit())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_LoadFile_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestConfigPath] = serviceTestConfigBody
	m.Files["/tmp/svc/.core/override.yaml"] = serviceTestShellYAML

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	core.AssertNoError(t, resultError(svc.LoadFile(m, serviceTestOverridePath)))

	var shell string
	core.AssertNoError(t, resultError(svc.Get(serviceTestDevShellKey, &shell)))
	core.AssertEqual(t, "zsh", shell)
}

func TestService_LoadFile_Bad_NoConfig(t *core.T) {
	svc := &Service{}

	err := resultError(svc.LoadFile(coreio.NewMockMedium(), serviceTestOverridePath))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_LoadFile_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestConfigPath] = serviceTestConfigBody

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	err := resultError(svc.LoadFile(m, core.PathJoin("tmp", "svc", "config.yaml")))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), serviceTestConfigPathsUnderCore)
}

func TestService_LoadFile_RejectsSymlinkedCore(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip(serviceTestWindowsSymlinkSkipMessage)
	}

	projectRoot := t.TempDir()
	externalCore := core.PathJoin(t.TempDir(), serviceTestSharedCoreDir)
	testMkdirAll(t, externalCore, 0755)
	testWriteFile(t, core.PathJoin(externalCore, serviceTestOverrideFilename), []byte(serviceTestShellYAML), 0600)
	testSymlink(t, externalCore, core.PathJoin(projectRoot, ".core"))
	testWriteFile(t, core.PathJoin(projectRoot, "config.yaml"), []byte(serviceTestConfigBody), 0600)

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   core.PathJoin(projectRoot, "config.yaml"),
			Medium: coreio.Local,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	err := resultError(svc.LoadFile(coreio.Local, serviceTestOverridePath))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), serviceTestSymlinkedCoreDirsRejected)
}

func TestService_resolveValidatedServiceLoadPath_Good(t *core.T) {
	projectRoot := t.TempDir()
	coreDir := core.PathJoin(projectRoot, ".core")
	configPath := core.PathJoin(projectRoot, "config.yaml")
	overridePath := core.PathJoin(coreDir, serviceTestOverrideFilename)

	testMkdirAll(t, coreDir, 0755)
	testWriteFile(t, configPath, []byte(serviceTestConfigBody), 0600)
	testWriteFile(t, overridePath, []byte("dev:\n  editor: vim\n"), 0600)

	resolved, err := stringResult(resolveValidatedServiceLoadPath(configPath, serviceTestOverridePath))
	core.AssertNoError(t, err)
	core.AssertEqual(t, overridePath, resolved)
}

func TestService_resolveValidatedServiceLoadPath_Bad(t *core.T) {
	projectRoot := t.TempDir()
	configPath := core.PathJoin(projectRoot, "config.yaml")
	testWriteFile(t, configPath, []byte(serviceTestConfigBody), 0600)

	cases := []struct {
		name string
		path string
		want string
	}{
		{name: "empty", path: "", want: "empty config path"},
		{name: "absolute", path: "/etc/passwd", want: "absolute config paths are not allowed"},
		{name: "traversal", path: "../escape.yaml", want: "path traversal rejected"},
		{name: "outside-core", path: "config.yaml", want: serviceTestConfigPathsUnderCore},
		{name: "nested-traversal", path: ".core/../escape.yaml", want: serviceTestConfigPathsUnderCore},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *core.T) {
			resolved, err := stringResult(resolveValidatedServiceLoadPath(configPath, tc.path))
			core.AssertEmpty(t, resolved)
			core.AssertError(t, err)
			core.AssertContains(t, err.Error(), tc.want)
		})
	}
}

func TestService_resolveValidatedServiceLoadPath_Ugly(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip(serviceTestWindowsSymlinkSkipMessage)
	}

	projectRoot := t.TempDir()
	externalCore := core.PathJoin(t.TempDir(), serviceTestSharedCoreDir)
	configPath := core.PathJoin(projectRoot, "config.yaml")

	testMkdirAll(t, externalCore, 0755)
	testWriteFile(t, configPath, []byte(serviceTestConfigBody), 0600)
	testSymlink(t, externalCore, core.PathJoin(projectRoot, ".core"))

	resolved, err := stringResult(resolveValidatedServiceLoadPath(configPath, serviceTestOverridePath))
	core.AssertEmpty(t, resolved)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), serviceTestSymlinkedCoreDirsRejected)
}

func TestService_resolveServiceLoadPath_Good(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip(serviceTestWindowsSymlinkSkipMessage)
	}

	projectRoot := t.TempDir()
	coreDir := core.PathJoin(projectRoot, ".core")
	realFile := core.PathJoin(projectRoot, serviceTestOverrideFilename)
	symlinkFile := core.PathJoin(coreDir, serviceTestOverrideFilename)

	testMkdirAll(t, coreDir, 0755)
	testWriteFile(t, realFile, []byte(serviceTestShellYAML), 0600)
	testSymlink(t, realFile, symlinkFile)

	absCorePath := testPathAbs(t, coreDir)
	absCandidate := testPathAbs(t, symlinkFile)

	resolvedCandidate, resolvedCore, err := serviceLoadPathResult(resolveServiceLoadPath(symlinkFile, absCorePath, absCandidate))
	core.AssertNoError(t, err)
	realCandidate := testPathEvalSymlinks(t, realFile)
	realCore := testPathEvalSymlinks(t, coreDir)
	core.AssertEqual(t, realCandidate, resolvedCandidate)
	core.AssertEqual(t, realCore, resolvedCore)
}

func TestService_resolveServiceLoadPath_Bad(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip(serviceTestWindowsSymlinkSkipMessage)
	}

	projectRoot := t.TempDir()
	externalCore := core.PathJoin(t.TempDir(), serviceTestSharedCoreDir)
	candidatePath := core.PathJoin(projectRoot, ".core", serviceTestOverrideFilename)

	testMkdirAll(t, externalCore, 0755)
	testSymlink(t, externalCore, core.PathJoin(projectRoot, ".core"))

	absCorePath := testPathAbs(t, core.PathJoin(projectRoot, ".core"))
	absCandidate := testPathAbs(t, candidatePath)

	resolvedCandidate, resolvedCore, err := serviceLoadPathResult(resolveServiceLoadPath(candidatePath, absCorePath, absCandidate))
	core.AssertEmpty(t, resolvedCandidate)
	core.AssertEmpty(t, resolvedCore)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), serviceTestSymlinkedCoreDirsRejected)
}

func TestService_OnShutdown_StopsWatcher_Good(t *core.T) {
	tmp := t.TempDir()
	path := core.PathJoin(tmp, "config.yaml")
	core.AssertNoError(t, coreio.Local.Write(path, serviceTestConfigBody))

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   path,
			Medium: coreio.Local,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)
	core.AssertNoError(t, resultError(svc.Config().Watch()))
	core.AssertNotNil(t, svc.Config().watcher)

	result := svc.OnShutdown(context.Background())
	core.AssertTrue(t, result.OK)
	core.AssertNil(t, svc.Config().watcher)
}

func TestService_Service_OnStartup_RegistersActionsAndCommands_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestConfigPath] = serviceTestConfigBody
	m.Files["/tmp/svc/.core/loaded.yaml"] = "dev:\n  editor: nano\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
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

	core.AssertTrue(t, runAction(serviceTestConfigGetAction, core.NewOptions(core.Option{Key: "key", Value: serviceTestAppNameKey})).OK)
	core.AssertTrue(t, runAction(serviceTestConfigSetAction, core.NewOptions(
		core.Option{Key: "key", Value: serviceTestDevShellKey},
		core.Option{Key: "value", Value: "zsh"},
	)).OK)
	core.AssertTrue(t, runAction(serviceTestConfigCommitAction, core.NewOptions()).OK)
	core.AssertTrue(t, runAction(serviceTestConfigLoadAction, core.NewOptions(core.Option{Key: optionKeyPath, Value: ".core/loaded.yaml"})).OK)

	all := runAction(serviceTestConfigAllAction, core.NewOptions())
	core.AssertTrue(t, all.OK)
	core.AssertContains(t, all.Value.(map[string]any), serviceTestAppNameKey)

	path := runAction(serviceTestConfigPathAction, core.NewOptions())
	core.AssertTrue(t, path.OK)
	core.AssertEqual(t, serviceTestConfigPath, path.Value)

	core.AssertTrue(t, runCommand(serviceTestConfigGetCommand, core.NewOptions(core.Option{Key: "key", Value: serviceTestAppNameKey})).OK)
	core.AssertTrue(t, runCommand(serviceTestConfigSetCommand, core.NewOptions(
		core.Option{Key: "key", Value: "dev.theme"},
		core.Option{Key: "value", Value: "dark"},
	)).OK)
	core.AssertTrue(t, runCommand(serviceTestConfigCommitCommand, core.NewOptions()).OK)
	core.AssertTrue(t, runCommand(serviceTestConfigLoadCommand, core.NewOptions(core.Option{Key: optionKeyPath, Value: ".core/loaded.yaml"})).OK)
	core.AssertTrue(t, runCommand(serviceTestConfigListCommand, core.NewOptions()).OK)
	core.AssertTrue(t, runCommand(serviceTestConfigPathCommand, core.NewOptions()).OK)
}

func TestServiceReadCommandsRequireEntitlement(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestConfigPath] = serviceTestConfigBody

	c := core.New()
	c.SetEntitlementChecker(func(action string, qty int, _ context.Context) core.Entitlement {
		_ = qty
		switch action {
		case serviceTestConfigGetCommand, serviceTestConfigListCommand, serviceTestConfigPathCommand:
			return core.Entitlement{Allowed: false, Reason: "denied"}
		default:
			return core.Entitlement{Allowed: true, Unlimited: true}
		}
	})

	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	for _, name := range []string{serviceTestConfigGetCommand, serviceTestConfigListCommand, serviceTestConfigPathCommand} {
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

func TestService_Service_OnStartup_ReadActionsRequireEntitlement_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[serviceTestConfigPath] = serviceTestConfigBody

	c := core.New()
	c.SetEntitlementChecker(func(action string, qty int, _ context.Context) core.Entitlement {
		_ = qty
		switch action {
		case serviceTestConfigGetAction, serviceTestConfigAllAction, serviceTestConfigPathAction:
			return core.Entitlement{Allowed: false, Reason: "denied"}
		default:
			return core.Entitlement{Allowed: true, Unlimited: true}
		}
	})

	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   serviceTestConfigPath,
			Medium: m,
		}),
	}
	core.AssertTrue(t, svc.OnStartup(context.Background()).OK)

	actions := map[string]core.Options{
		serviceTestConfigGetAction:  core.NewOptions(core.Option{Key: "key", Value: serviceTestAppNameKey}),
		serviceTestConfigAllAction:  core.NewOptions(),
		serviceTestConfigPathAction: core.NewOptions(),
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
	m.Files[path] = serviceTestConfigBody
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

func TestService_Service_OnStartup_LoadsConfig_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := "/ax7/service/config.yaml"
	m.Files[path] = serviceTestConfigBody
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{Path: path, Medium: m})}
	core.RequireTrue(t, svc.OnStartup(context.Background()).OK)
	var got string
	err := resultError(svc.Get(serviceTestAppNameKey, &got))
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
	svc, _, path := axServiceFixture(t)
	var got string
	err := resultError(svc.Get(serviceTestAppNameKey, &got))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "svc", got)
	core.AssertEqual(t, path, svc.Options().Path)
}

func TestService_Service_Get_Bad(t *core.T) {
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{})}
	err := resultError(svc.Get("missing", new(string)))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_Service_Get_Ugly(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	var got map[string]any
	err := resultError(svc.Get("", &got))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "svc", got["app"].(map[string]any)["name"])
}

func TestService_Service_Set_Good(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	err := resultError(svc.Set(serviceTestDevEditorKey, "vim"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "vim", configValues(svc.Config())[serviceTestDevEditorKey])
}

func TestService_Service_Set_Bad(t *core.T) {
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{})}
	err := resultError(svc.Set(serviceTestDevEditorKey, "vim"))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_Service_Set_Ugly(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	err := resultError(svc.Set("", "root"))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "root", svc.Config().file.Get(""))
}

func TestService_Service_Commit_Good(t *core.T) {
	svc, m, path := axServiceFixture(t)
	core.RequireNoError(t, resultError(svc.Set(serviceTestDevEditorKey, "vim")))
	err := resultError(svc.Commit())
	core.AssertNoError(t, err)
	core.AssertTrue(t, m.Exists(path))
}

func TestService_Service_Commit_Bad(t *core.T) {
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{})}
	err := resultError(svc.Commit())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_Service_Commit_Ugly(t *core.T) {
	svc, _, _ := axServiceFixture(t)
	svc.Config().path = "/ax7/config.json"
	err := resultError(svc.Commit())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsupported config file type")
}

func TestService_Service_LoadFile_Good(t *core.T) {
	svc, m, _ := axServiceFixture(t)
	core.RequireNoError(t, m.Write("/ax7/service/.core/override.yaml", serviceTestShellYAML))
	err := resultError(svc.LoadFile(m, serviceTestOverridePath))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "zsh", configValues(svc.Config())[serviceTestDevShellKey])
}

func TestService_Service_LoadFile_Bad(t *core.T) {
	svc := &Service{ServiceRuntime: core.NewServiceRuntime(core.New(), ServiceOptions{})}
	err := resultError(svc.LoadFile(coreio.NewMockMedium(), serviceTestOverridePath))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), errConfigNotLoaded)
}

func TestService_Service_LoadFile_Ugly(t *core.T) {
	svc, m, _ := axServiceFixture(t)
	err := resultError(svc.LoadFile(m, "../escape.yaml"))
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
	replacement, err := configResult(New(WithMedium(coreio.NewMockMedium()), WithPath("/ax7/replacement.yaml")))
	core.RequireNoError(t, err)
	svc.config = replacement
	core.AssertSame(t, replacement, svc.Config())
}
