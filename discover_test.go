package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func TestDiscover_DiscoverFrom_Good(t *core.T) {
	m := coreio.NewMockMedium()
	repo := core.Path("repo")
	sub := core.Path(repo, "service")
	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".core")))
	core.AssertNoError(t, m.EnsureDir(core.Path(sub, ".core")))
	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".git")))
	core.AssertNoError(t, m.EnsureDir(sub))

	core.AssertNoError(t, m.Write(core.Path(repo, ".core", "config.yaml"), "dev:\n  editor: vim\napp:\n  name: repo\n"))
	core.AssertNoError(t, m.Write(core.Path(sub, ".core", "config.yaml"), "app:\n  name: service\n"))

	cfg, err := DiscoverFrom(sub, WithMedium(m), WithPath(core.Path(sub, ".core", "config.yaml")))
	core.AssertNoError(t, err)

	// Closest (service) wins on app.name.
	var name string
	core.AssertNoError(t, cfg.Get("app.name", &name))
	core.AssertEqual(t, "service", name)

	// Parent fills the gap on dev.editor.
	var editor string
	core.AssertNoError(t, cfg.Get("dev.editor", &editor))
	core.AssertEqual(t, "vim", editor)
}

func TestDiscover_DiscoverFrom_Bad(t *core.T) {
	// A .core/config.yaml with malformed YAML makes the layered load fail.
	m := coreio.NewMockMedium()
	root := core.Path("bad-repo")
	core.AssertNoError(t, m.EnsureDir(core.Path(root, ".core")))
	core.AssertNoError(t, m.Write(core.Path(root, ".core", "config.yaml"), "invalid: [yaml"))

	_, err := DiscoverFrom(root, WithMedium(m))
	core.AssertError(t, err)
}

func TestDiscover_DiscoverFrom_Ugly(t *core.T) {
	// Empty start directory — uses filesystem root walk, should still return a
	// usable (but empty) config rather than panicking.
	m := coreio.NewMockMedium()
	cfg, err := DiscoverFrom("/nonexistent/path", WithMedium(m), WithPath("/nonexistent/path/config.yaml"))
	core.AssertNoError(t, err)
	core.AssertNotNil(t, cfg)
}

func TestDiscover_CoreDirs_Good(t *core.T) {
	m := coreio.NewMockMedium()
	repo := core.Path("dirs-repo")
	sub := core.Path(repo, "service")
	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".core")))
	core.AssertNoError(t, m.EnsureDir(core.Path(sub, ".core")))
	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".git")))
	core.AssertNoError(t, m.EnsureDir(sub))

	dirs := CoreDirs(m, sub)
	// Closest first: sub/.core, then repo/.core. Walk stops at repo (.git boundary).
	core.AssertGreaterOrEqual(t, len(dirs), 2)
	core.AssertEqual(t, core.Path(sub, ".core"), dirs[0])
	core.AssertEqual(t, core.Path(repo, ".core"), dirs[1])
}

func TestDiscover_CoreDirs_Bad(t *core.T) {
	// A directory tree with no .core anywhere just returns the home layer (if any).
	m := coreio.NewMockMedium()
	root := core.Path("empty-repo")
	core.AssertNoError(t, m.EnsureDir(root))
	dirs := CoreDirs(m, root)
	for _, dir := range dirs {
		core.AssertNotContains(t, dir, root)
	}
}

func TestDiscover_FindManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	repo := core.Path("manifest-repo")
	sub := core.Path(repo, "service")
	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".core")))
	core.AssertNoError(t, m.EnsureDir(core.Path(sub, ".core")))
	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".git")))
	core.AssertNoError(t, m.EnsureDir(sub))
	// Only the repo-level .core/ has build.yaml.
	core.AssertNoError(t, m.Write(core.Path(repo, ".core", "build.yaml"), "name: core\n"))

	path := FindManifest(m, sub, FileBuild)
	core.AssertEqual(t, core.Path(repo, ".core", "build.yaml"), path)
}

func TestDiscover_FindManifest_Ugly(t *core.T) {
	// Missing file returns empty string, not an error.
	m := coreio.NewMockMedium()
	start := core.Path("missing-repo")
	got := FindManifest(m, start, FileBuild)
	core.AssertEmpty(t, got)
}

func TestDiscoverEnvOverridesDiscoveredGood(t *core.T) {
	// .core/ convention §5.3: "Env vars override everything."
	// A discovered file value must be shadowed by CORE_CONFIG_* at Get time.
	m := coreio.NewMockMedium()
	root := core.Path("env-repo")
	t.Setenv("CORE_CONFIG_APP_NAME", "env-wins")

	core.AssertNoError(t, m.EnsureDir(core.Path(root, ".core")))
	core.AssertNoError(t, m.Write(
		core.Path(root, ".core", "config.yaml"),
		"app:\n  name: fromfile\n",
	))

	cfg, err := DiscoverFrom(root, WithMedium(m))
	core.AssertNoError(t, err)

	var name string
	core.AssertNoError(t, cfg.Get("app.name", &name))
	core.AssertEqual(t, "env-wins", name)
}

func TestDiscoverMergeFillsGapsGood(t *core.T) {
	// Project .core/ wins over global .core/ — global only fills gaps.
	m := coreio.NewMockMedium()
	repo := core.Path("merge-repo")
	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".core")))
	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".git")))
	core.AssertNoError(t, m.Write(
		core.Path(repo, ".core", "config.yaml"),
		"app:\n  name: project\n",
	))

	cfg, err := DiscoverFrom(repo, WithMedium(m))
	core.AssertNoError(t, err)

	var name string
	core.AssertNoError(t, cfg.Get("app.name", &name))
	core.AssertEqual(t, "project", name)
}

func TestDiscoverCommitDoesNotLeakInheritedGood(t *core.T) {
	// Regression guard: Commit on a discovered Config must only persist the
	// owning file's keys + Set() calls, never inherited layer values — or
	// global ~/.core/ secrets would spray into every project config.
	m := coreio.NewMockMedium()
	repo := core.Path("commit-repo")
	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".core")))
	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".git")))
	core.AssertNoError(t, m.Write(
		core.Path(repo, ".core", "config.yaml"),
		"secret:\n  token: GLOBAL_ONLY\n",
	))

	commitPath := core.Path("commit-repo", "newcfg.yaml")
	cfg, err := DiscoverFrom(repo, WithMedium(m), WithPath(commitPath))
	core.AssertNoError(t, err)

	// Set our own key then Commit; the inherited GLOBAL_ONLY must NOT appear.
	core.AssertNoError(t, cfg.Set("dev.shell", "zsh"))
	core.AssertNoError(t, cfg.Commit())

	body, err := m.Read(commitPath)
	core.AssertNoError(t, err)
	core.AssertContains(t, body, "dev:")
	core.AssertContains(t, body, "shell: zsh")
	core.AssertNotContains(t, body, "GLOBAL_ONLY")
	core.AssertNotContains(t, body, "secret:")
}

func TestDiscover_DiscoverFrom_GlobalFallback_Good(t *core.T) {
	m := coreio.NewMockMedium()
	repo := core.Path("global-repo")
	home := core.Env("DIR_HOME")

	core.AssertNoError(t, m.EnsureDir(core.Path(repo, ".git")))
	core.AssertNoError(t, m.EnsureDir(core.Path(home, ".core")))
	core.AssertNoError(t, m.Write(core.Path(home, ".core", "config.yaml"), "app:\n  name: global\n"))

	cfg, err := DiscoverFrom(repo, WithMedium(m))
	core.AssertNoError(t, err)

	var name string
	core.AssertNoError(t, cfg.Get("app.name", &name))
	core.AssertEqual(t, "global", name)
}

func TestDiscover_Discover_Good(t *core.T) {
	root := t.TempDir()
	coreDir := core.PathJoin(root, ".core")
	testMkdirAll(t, coreDir, 0o755)
	testWriteFile(t, core.PathJoin(coreDir, FileConfig), []byte("app:\n  name: discovered\n"), 0o600)
	previous := testGetwd(t)
	testChdir(t, root)
	t.Cleanup(func() { testChdir(t, previous) })

	cfg, err := Discover()
	core.RequireNoError(t, err)
	var got string
	core.AssertNoError(t, cfg.Get("app.name", &got))
	core.AssertEqual(t, "discovered", got)
}

func TestDiscover_Discover_Bad(t *core.T) {
	root := t.TempDir()
	coreDir := core.PathJoin(root, ".core")
	testMkdirAll(t, coreDir, 0o755)
	testWriteFile(t, core.PathJoin(coreDir, FileConfig), []byte("bad: [yaml"), 0o600)
	previous := testGetwd(t)
	testChdir(t, root)
	t.Cleanup(func() { testChdir(t, previous) })

	cfg, err := Discover()
	core.AssertNil(t, cfg)
	core.AssertError(t, err)
}

func TestDiscover_Discover_Ugly(t *core.T) {
	root := t.TempDir()
	previous := testGetwd(t)
	testChdir(t, root)
	t.Cleanup(func() { testChdir(t, previous) })

	cfg, err := Discover()
	core.RequireNoError(t, err)
	core.AssertError(t, cfg.Get("missing", new(string)))
}

func TestDiscover_CoreDirs_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	start := core.Path("lonely", "service")
	got := CoreDirs(m, start)
	core.AssertEmpty(t, got)
}

func TestDiscover_FindManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindManifest(m, core.Path("repo"), "../config.yaml")
	core.AssertEqual(t, "", got)
}
