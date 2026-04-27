package config

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	"github.com/stretchr/testify/assert"
)

func TestService_OnStartup_Good(t *testing.T) {
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
	assert.True(t, result.OK)

	var name string
	assert.NoError(t, svc.Get("app.name", &name))
	assert.Equal(t, "svc", name)
}

func TestService_OnStartup_Bad(t *testing.T) {
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
	assert.False(t, result.OK)
}

func TestService_OnStartup_RegistersActions_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "dev:\n  editor: vim\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	assert.True(t, svc.OnStartup(context.Background()).OK)

	// config.get must round-trip through the action bus.
	result := c.Action("config.get").Run(context.Background(), core.NewOptions(core.Option{Key: "key", Value: "dev.editor"}))
	assert.True(t, result.OK)
	assert.Equal(t, "vim", result.Value)

	// config.set stores a value; config.get reads it back.
	setResult := c.Action("config.set").Run(context.Background(), core.NewOptions(
		core.Option{Key: "key", Value: "dev.shell"},
		core.Option{Key: "value", Value: "zsh"},
	))
	assert.True(t, setResult.OK)

	readResult := c.Action("config.get").Run(context.Background(), core.NewOptions(core.Option{Key: "key", Value: "dev.shell"}))
	assert.True(t, readResult.OK)
	assert.Equal(t, "zsh", readResult.Value)
}

func TestService_OnStartup_RegistersCommands_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	assert.True(t, svc.OnStartup(context.Background()).OK)

	assert.Contains(t, c.Commands(), "config/get")
	assert.Contains(t, c.Commands(), "config/set")
	assert.Contains(t, c.Commands(), "config/list")
}

func TestService_OnStartup_MergesProjectOverGlobal_Good(t *testing.T) {
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
		assert.NoError(t, m.EnsureDir(dir))
	}

	assert.NoError(t, m.Write(filepath.Join(home, ".core", FileConfig), "app:\n  name: global\nservices:\n  ollama:\n    url: http://global\n"))
	assert.NoError(t, m.Write(filepath.Join(projectRoot, ".core", FileConfig), "app:\n  name: project\n"))

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   filepath.Join(projectRoot, ".core", FileConfig),
			Medium: m,
		}),
	}

	assert.True(t, svc.OnStartup(context.Background()).OK)

	var name string
	assert.NoError(t, svc.Get("app.name", &name))
	assert.Equal(t, "project", name)

	var ollamaURL string
	assert.NoError(t, svc.Get("services.ollama.url", &ollamaURL))
	assert.Equal(t, "http://global", ollamaURL)
}

func TestService_Config_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}

	// Before OnStartup, Config() returns nil.
	assert.Nil(t, svc.Config())

	assert.True(t, svc.OnStartup(context.Background()).OK)
	assert.NotNil(t, svc.Config())
}

func TestService_Get_Bad(t *testing.T) {
	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{}),
	}
	var v string
	err := svc.Get("anything", &v)
	assert.Error(t, err)
}

func TestService_NewConfigService_Good(t *testing.T) {
	// The documented usage must compile and succeed: the factory is a
	// core.WithService value and produces a retrievable *Service.
	c := core.New(core.WithService(NewConfigService))
	svc, ok := core.ServiceFor[*Service](c, "config")
	assert.True(t, ok)
	assert.NotNil(t, svc)
}

func TestService_NewConfigServiceWith_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/custom/config.yaml"] = "app:\n  name: custom\n"

	c := core.New(core.WithService(NewConfigServiceWith(ServiceOptions{
		Path:   "/tmp/custom/config.yaml",
		Medium: m,
	})))

	svc, ok := core.ServiceFor[*Service](c, "config")
	assert.True(t, ok)

	// OnStartup has not run yet; trigger it via the Core's service lifecycle.
	startables := c.Startables()
	assert.True(t, startables.OK)
	for _, s := range startables.Value.([]*core.Service) {
		assert.True(t, s.OnStart().OK)
	}

	var name string
	assert.NoError(t, svc.Get("app.name", &name))
	assert.Equal(t, "custom", name)
}

func TestService_NewConfigService_Bad(t *testing.T) {
	// With a broken medium path (unsupported file type) OnStartup must fail
	// gracefully and return a non-OK Result rather than panicking.
	m := coreio.NewMockMedium()
	m.Files["/broken.txt"] = "ignored"
	c := core.New(core.WithService(NewConfigServiceWith(ServiceOptions{
		Path:   "/broken.txt",
		Medium: m,
	})))
	svc, ok := core.ServiceFor[*Service](c, "config")
	assert.True(t, ok)

	startables := c.Startables()
	assert.True(t, startables.OK)
	var gotFailure bool
	for _, s := range startables.Value.([]*core.Service) {
		if !s.OnStart().OK {
			gotFailure = true
		}
	}
	assert.True(t, gotFailure)
	assert.Nil(t, svc.Config())
}

func TestService_LoadFile_RejectsUnsafePaths(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	assert.True(t, svc.OnStartup(context.Background()).OK)

	err := svc.LoadFile(m, "../../etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "path traversal rejected")

	err = svc.LoadFile(m, "/etc/passwd")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "absolute config paths are not allowed")
}

func TestService_Set_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	assert.True(t, svc.OnStartup(context.Background()).OK)

	assert.NoError(t, svc.Set("dev.editor", "vim"))

	var editor string
	assert.NoError(t, svc.Get("dev.editor", &editor))
	assert.Equal(t, "vim", editor)
}

func TestService_Set_Bad(t *testing.T) {
	svc := &Service{}

	err := svc.Set("dev.editor", "vim")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not loaded")
}

func TestService_Commit_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	assert.True(t, svc.OnStartup(context.Background()).OK)
	assert.NoError(t, svc.Set("dev.editor", "vim"))

	assert.NoError(t, svc.Commit())
	body, err := m.Read("/tmp/svc/config.yaml")
	assert.NoError(t, err)
	assert.Contains(t, body, "editor: vim")
}

func TestService_Commit_Bad(t *testing.T) {
	svc := &Service{}

	err := svc.Commit()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not loaded")
}

func TestService_LoadFile_Good(t *testing.T) {
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
	assert.True(t, svc.OnStartup(context.Background()).OK)

	assert.NoError(t, svc.LoadFile(m, ".core/override.yaml"))

	var shell string
	assert.NoError(t, svc.Get("dev.shell", &shell))
	assert.Equal(t, "zsh", shell)
}

func TestService_LoadFile_Bad_NoConfig(t *testing.T) {
	svc := &Service{}

	err := svc.LoadFile(coreio.NewMockMedium(), ".core/override.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config not loaded")
}

func TestService_LoadFile_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/tmp/svc/config.yaml"] = "app:\n  name: svc\n"

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   "/tmp/svc/config.yaml",
			Medium: m,
		}),
	}
	assert.True(t, svc.OnStartup(context.Background()).OK)

	err := svc.LoadFile(m, filepath.Join("tmp", "svc", "config.yaml"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "config paths must remain under .core/")
}

func TestService_LoadFile_RejectsSymlinkedCore(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	projectRoot := t.TempDir()
	externalCore := filepath.Join(t.TempDir(), "shared-core")
	assert.NoError(t, os.MkdirAll(externalCore, 0755))
	assert.NoError(t, os.WriteFile(filepath.Join(externalCore, "override.yaml"), []byte("dev:\n  shell: zsh\n"), 0600))
	assert.NoError(t, os.Symlink(externalCore, filepath.Join(projectRoot, ".core")))
	assert.NoError(t, os.WriteFile(filepath.Join(projectRoot, "config.yaml"), []byte("app:\n  name: svc\n"), 0600))

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   filepath.Join(projectRoot, "config.yaml"),
			Medium: coreio.Local,
		}),
	}
	assert.True(t, svc.OnStartup(context.Background()).OK)

	err := svc.LoadFile(coreio.Local, ".core/override.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symlinked .core directories are not allowed")
}

func TestService_ResolveValidatedServiceLoadPath_Good(t *testing.T) {
	projectRoot := t.TempDir()
	coreDir := filepath.Join(projectRoot, ".core")
	configPath := filepath.Join(projectRoot, "config.yaml")
	overridePath := filepath.Join(coreDir, "override.yaml")

	assert.NoError(t, os.MkdirAll(coreDir, 0755))
	assert.NoError(t, os.WriteFile(configPath, []byte("app:\n  name: svc\n"), 0600))
	assert.NoError(t, os.WriteFile(overridePath, []byte("dev:\n  editor: vim\n"), 0600))

	resolved, err := resolveValidatedServiceLoadPath(configPath, ".core/override.yaml")
	assert.NoError(t, err)
	assert.Equal(t, overridePath, resolved)
}

func TestService_ResolveValidatedServiceLoadPath_Bad(t *testing.T) {
	projectRoot := t.TempDir()
	configPath := filepath.Join(projectRoot, "config.yaml")
	assert.NoError(t, os.WriteFile(configPath, []byte("app:\n  name: svc\n"), 0600))

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
		t.Run(tc.name, func(t *testing.T) {
			resolved, err := resolveValidatedServiceLoadPath(configPath, tc.path)
			assert.Empty(t, resolved)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestService_ResolveValidatedServiceLoadPath_Ugly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	projectRoot := t.TempDir()
	externalCore := filepath.Join(t.TempDir(), "shared-core")
	configPath := filepath.Join(projectRoot, "config.yaml")

	assert.NoError(t, os.MkdirAll(externalCore, 0755))
	assert.NoError(t, os.WriteFile(configPath, []byte("app:\n  name: svc\n"), 0600))
	assert.NoError(t, os.Symlink(externalCore, filepath.Join(projectRoot, ".core")))

	resolved, err := resolveValidatedServiceLoadPath(configPath, ".core/override.yaml")
	assert.Empty(t, resolved)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symlinked .core directories are not allowed")
}

func TestService_ResolveServiceLoadPath_Good(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	projectRoot := t.TempDir()
	coreDir := filepath.Join(projectRoot, ".core")
	realFile := filepath.Join(projectRoot, "override.yaml")
	symlinkFile := filepath.Join(coreDir, "override.yaml")

	assert.NoError(t, os.MkdirAll(coreDir, 0755))
	assert.NoError(t, os.WriteFile(realFile, []byte("dev:\n  shell: zsh\n"), 0600))
	assert.NoError(t, os.Symlink(realFile, symlinkFile))

	absCorePath, err := filepath.Abs(coreDir)
	assert.NoError(t, err)
	absCandidate, err := filepath.Abs(symlinkFile)
	assert.NoError(t, err)

	resolvedCandidate, resolvedCore, err := resolveServiceLoadPath(symlinkFile, absCorePath, absCandidate)
	assert.NoError(t, err)
	realCandidate, err := filepath.EvalSymlinks(realFile)
	assert.NoError(t, err)
	realCore, err := filepath.EvalSymlinks(coreDir)
	assert.NoError(t, err)
	assert.Equal(t, realCandidate, resolvedCandidate)
	assert.Equal(t, realCore, resolvedCore)
}

func TestService_ResolveServiceLoadPath_Bad(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	projectRoot := t.TempDir()
	externalCore := filepath.Join(t.TempDir(), "shared-core")
	candidatePath := filepath.Join(projectRoot, ".core", "override.yaml")

	assert.NoError(t, os.MkdirAll(externalCore, 0755))
	assert.NoError(t, os.Symlink(externalCore, filepath.Join(projectRoot, ".core")))

	absCorePath, err := filepath.Abs(filepath.Join(projectRoot, ".core"))
	assert.NoError(t, err)
	absCandidate, err := filepath.Abs(candidatePath)
	assert.NoError(t, err)

	resolvedCandidate, resolvedCore, err := resolveServiceLoadPath(candidatePath, absCorePath, absCandidate)
	assert.Empty(t, resolvedCandidate)
	assert.Empty(t, resolvedCore)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symlinked .core directories are not allowed")
}

func TestService_OnShutdown_StopsWatcher_Good(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	assert.NoError(t, coreio.Local.Write(path, "app:\n  name: svc\n"))

	c := core.New()
	svc := &Service{
		ServiceRuntime: core.NewServiceRuntime(c, ServiceOptions{
			Path:   path,
			Medium: coreio.Local,
		}),
	}
	assert.True(t, svc.OnStartup(context.Background()).OK)
	assert.NoError(t, svc.Config().Watch())
	assert.NotNil(t, svc.Config().watcher)

	result := svc.OnShutdown(context.Background())
	assert.True(t, result.OK)
	assert.Nil(t, svc.Config().watcher)
}

func TestService_RegistersActionsAndCommands_Good(t *testing.T) {
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
	assert.True(t, svc.OnStartup(context.Background()).OK)

	runAction := func(name string, opts core.Options) core.Result {
		return c.Action(name).Run(context.Background(), opts)
	}
	runCommand := func(name string, opts core.Options) core.Result {
		r := c.Command(name)
		if !assert.True(t, r.OK) {
			return core.Result{}
		}
		return r.Value.(*core.Command).Run(opts)
	}

	assert.True(t, runAction("config.get", core.NewOptions(core.Option{Key: "key", Value: "app.name"})).OK)
	assert.True(t, runAction("config.set", core.NewOptions(
		core.Option{Key: "key", Value: "dev.shell"},
		core.Option{Key: "value", Value: "zsh"},
	)).OK)
	assert.True(t, runAction("config.commit", core.NewOptions()).OK)
	assert.True(t, runAction("config.load", core.NewOptions(core.Option{Key: "path", Value: ".core/loaded.yaml"})).OK)

	all := runAction("config.all", core.NewOptions())
	assert.True(t, all.OK)
	assert.Contains(t, all.Value.(map[string]any), "app.name")

	path := runAction("config.path", core.NewOptions())
	assert.True(t, path.OK)
	assert.Equal(t, "/tmp/svc/config.yaml", path.Value)

	assert.True(t, runCommand("config/get", core.NewOptions(core.Option{Key: "key", Value: "app.name"})).OK)
	assert.True(t, runCommand("config/set", core.NewOptions(
		core.Option{Key: "key", Value: "dev.theme"},
		core.Option{Key: "value", Value: "dark"},
	)).OK)
	assert.True(t, runCommand("config/commit", core.NewOptions()).OK)
	assert.True(t, runCommand("config/load", core.NewOptions(core.Option{Key: "path", Value: ".core/loaded.yaml"})).OK)
	assert.True(t, runCommand("config/list", core.NewOptions()).OK)
	assert.True(t, runCommand("config/path", core.NewOptions()).OK)
}

func TestService_ReadCommands_RequireEntitlement(t *testing.T) {
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
	assert.True(t, svc.OnStartup(context.Background()).OK)

	for _, name := range []string{"config/get", "config/list", "config/path"} {
		cmdResult := c.Command(name)
		if !assert.True(t, cmdResult.OK, name) {
			continue
		}
		res := cmdResult.Value.(*core.Command).Run(core.NewOptions())
		assert.False(t, res.OK, name)
		assert.Contains(t, res.Value.(error).Error(), "not entitled")
	}
}

func TestService_ReadActions_RequireEntitlement_Bad(t *testing.T) {
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
	assert.True(t, svc.OnStartup(context.Background()).OK)

	actions := map[string]core.Options{
		"config.get":  core.NewOptions(core.Option{Key: "key", Value: "app.name"}),
		"config.all":  core.NewOptions(),
		"config.path": core.NewOptions(),
	}

	for name, opts := range actions {
		t.Run(name, func(t *testing.T) {
			res := c.Action(name).Run(context.Background(), opts)
			assert.False(t, res.OK)
			assert.Contains(t, res.Value.(error).Error(), "not entitled")
		})
	}
}
