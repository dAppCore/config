package config

import (
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	"github.com/stretchr/testify/assert"
)

func TestDiscover_DiscoverFrom_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	repo := core.Path("repo")
	sub := core.Path(repo, "service")
	assert.NoError(t, m.EnsureDir(core.Path(repo, ".core")))
	assert.NoError(t, m.EnsureDir(core.Path(sub, ".core")))
	assert.NoError(t, m.EnsureDir(core.Path(repo, ".git")))
	assert.NoError(t, m.EnsureDir(sub))

	assert.NoError(t, m.Write(core.Path(repo, ".core", "config.yaml"), "dev:\n  editor: vim\napp:\n  name: repo\n"))
	assert.NoError(t, m.Write(core.Path(sub, ".core", "config.yaml"), "app:\n  name: service\n"))

	cfg, err := DiscoverFrom(sub, WithMedium(m), WithPath(core.Path(sub, ".core", "config.yaml")))
	assert.NoError(t, err)

	// Closest (service) wins on app.name.
	var name string
	assert.NoError(t, cfg.Get("app.name", &name))
	assert.Equal(t, "service", name)

	// Parent fills the gap on dev.editor.
	var editor string
	assert.NoError(t, cfg.Get("dev.editor", &editor))
	assert.Equal(t, "vim", editor)
}

func TestDiscover_DiscoverFrom_Bad(t *testing.T) {
	// A .core/config.yaml with malformed YAML makes the layered load fail.
	m := coreio.NewMockMedium()
	root := core.Path("bad-repo")
	assert.NoError(t, m.EnsureDir(core.Path(root, ".core")))
	assert.NoError(t, m.Write(core.Path(root, ".core", "config.yaml"), "invalid: [yaml"))

	_, err := DiscoverFrom(root, WithMedium(m))
	assert.Error(t, err)
}

func TestDiscover_DiscoverFrom_Ugly(t *testing.T) {
	// Empty start directory — uses filesystem root walk, should still return a
	// usable (but empty) config rather than panicking.
	m := coreio.NewMockMedium()
	cfg, err := DiscoverFrom("/nonexistent/path", WithMedium(m), WithPath("/nonexistent/path/config.yaml"))
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestDiscover_CoreDirs_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	repo := core.Path("dirs-repo")
	sub := core.Path(repo, "service")
	assert.NoError(t, m.EnsureDir(core.Path(repo, ".core")))
	assert.NoError(t, m.EnsureDir(core.Path(sub, ".core")))
	assert.NoError(t, m.EnsureDir(core.Path(repo, ".git")))
	assert.NoError(t, m.EnsureDir(sub))

	dirs := CoreDirs(m, sub)
	// Closest first: sub/.core, then repo/.core. Walk stops at repo (.git boundary).
	assert.GreaterOrEqual(t, len(dirs), 2)
	assert.Equal(t, core.Path(sub, ".core"), dirs[0])
	assert.Equal(t, core.Path(repo, ".core"), dirs[1])
}

func TestDiscover_CoreDirs_Bad(t *testing.T) {
	// A directory tree with no .core anywhere just returns the home layer (if any).
	m := coreio.NewMockMedium()
	root := core.Path("empty-repo")
	assert.NoError(t, m.EnsureDir(root))
	dirs := CoreDirs(m, root)
	for _, dir := range dirs {
		assert.NotContains(t, dir, root)
	}
}

func TestDiscover_FindManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	repo := core.Path("manifest-repo")
	sub := core.Path(repo, "service")
	assert.NoError(t, m.EnsureDir(core.Path(repo, ".core")))
	assert.NoError(t, m.EnsureDir(core.Path(sub, ".core")))
	assert.NoError(t, m.EnsureDir(core.Path(repo, ".git")))
	assert.NoError(t, m.EnsureDir(sub))
	// Only the repo-level .core/ has build.yaml.
	assert.NoError(t, m.Write(core.Path(repo, ".core", "build.yaml"), "name: core\n"))

	path := FindManifest(m, sub, FileBuild)
	assert.Equal(t, core.Path(repo, ".core", "build.yaml"), path)
}

func TestDiscover_FindManifest_Ugly(t *testing.T) {
	// Missing file returns empty string, not an error.
	m := coreio.NewMockMedium()
	assert.Empty(t, FindManifest(m, core.Path("missing-repo"), FileBuild))
}

func TestDiscover_EnvOverridesDiscovered_Good(t *testing.T) {
	// .core/ convention §5.3: "Env vars override everything."
	// A discovered file value must be shadowed by CORE_CONFIG_* at Get time.
	m := coreio.NewMockMedium()
	root := core.Path("env-repo")
	t.Setenv("CORE_CONFIG_APP_NAME", "env-wins")

	assert.NoError(t, m.EnsureDir(core.Path(root, ".core")))
	assert.NoError(t, m.Write(
		core.Path(root, ".core", "config.yaml"),
		"app:\n  name: fromfile\n",
	))

	cfg, err := DiscoverFrom(root, WithMedium(m))
	assert.NoError(t, err)

	var name string
	assert.NoError(t, cfg.Get("app.name", &name))
	assert.Equal(t, "env-wins", name)
}

func TestDiscover_MergeFillsGaps_Good(t *testing.T) {
	// Project .core/ wins over global .core/ — global only fills gaps.
	m := coreio.NewMockMedium()
	repo := core.Path("merge-repo")
	assert.NoError(t, m.EnsureDir(core.Path(repo, ".core")))
	assert.NoError(t, m.EnsureDir(core.Path(repo, ".git")))
	assert.NoError(t, m.Write(
		core.Path(repo, ".core", "config.yaml"),
		"app:\n  name: project\n",
	))

	cfg, err := DiscoverFrom(repo, WithMedium(m))
	assert.NoError(t, err)

	var name string
	assert.NoError(t, cfg.Get("app.name", &name))
	assert.Equal(t, "project", name)
}

func TestDiscover_CommitDoesNotLeakInherited_Good(t *testing.T) {
	// Regression guard: Commit on a discovered Config must only persist the
	// owning file's keys + Set() calls, never inherited layer values — or
	// global ~/.core/ secrets would spray into every project config.
	m := coreio.NewMockMedium()
	repo := core.Path("commit-repo")
	assert.NoError(t, m.EnsureDir(core.Path(repo, ".core")))
	assert.NoError(t, m.EnsureDir(core.Path(repo, ".git")))
	assert.NoError(t, m.Write(
		core.Path(repo, ".core", "config.yaml"),
		"secret:\n  token: GLOBAL_ONLY\n",
	))

	commitPath := core.Path("commit-repo", "newcfg.yaml")
	cfg, err := DiscoverFrom(repo, WithMedium(m), WithPath(commitPath))
	assert.NoError(t, err)

	// Set our own key then Commit; the inherited GLOBAL_ONLY must NOT appear.
	assert.NoError(t, cfg.Set("dev.shell", "zsh"))
	assert.NoError(t, cfg.Commit())

	body, err := m.Read(commitPath)
	assert.NoError(t, err)
	assert.Contains(t, body, "dev:")
	assert.Contains(t, body, "shell: zsh")
	assert.NotContains(t, body, "GLOBAL_ONLY")
	assert.NotContains(t, body, "secret:")
}

func TestDiscover_DiscoverFrom_GlobalFallback_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	repo := core.Path("global-repo")
	home := core.Env("DIR_HOME")

	assert.NoError(t, m.EnsureDir(core.Path(repo, ".git")))
	assert.NoError(t, m.EnsureDir(core.Path(home, ".core")))
	assert.NoError(t, m.Write(core.Path(home, ".core", "config.yaml"), "app:\n  name: global\n"))

	cfg, err := DiscoverFrom(repo, WithMedium(m))
	assert.NoError(t, err)

	var name string
	assert.NoError(t, cfg.Get("app.name", &name))
	assert.Equal(t, "global", name)
}
