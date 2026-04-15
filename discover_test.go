package config

import (
	"path/filepath"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

func TestDiscover_DiscoverFrom_Good(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	sub := filepath.Join(repo, "service")
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(sub, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".git")))

	assert.NoError(t, coreio.Local.Write(filepath.Join(repo, ".core", "config.yaml"), "dev:\n  editor: vim\napp:\n  name: repo\n"))
	assert.NoError(t, coreio.Local.Write(filepath.Join(sub, ".core", "config.yaml"), "app:\n  name: service\n"))

	cfg, err := DiscoverFrom(sub, WithMedium(coreio.Local), WithPath(filepath.Join(sub, ".core", "config.yaml")))
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
	tmp := t.TempDir()
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(tmp, ".core")))
	assert.NoError(t, coreio.Local.Write(filepath.Join(tmp, ".core", "config.yaml"), "invalid: [yaml"))

	_, err := DiscoverFrom(tmp, WithMedium(coreio.Local))
	assert.Error(t, err)
}

func TestDiscover_DiscoverFrom_Ugly(t *testing.T) {
	// Empty start directory — uses filesystem root walk, should still return a
	// usable (but empty) config rather than panicking.
	cfg, err := DiscoverFrom("/nonexistent/path", WithMedium(coreio.Local), WithPath("/nonexistent/path/config.yaml"))
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestDiscover_CoreDirs_Good(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	sub := filepath.Join(repo, "service")
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(sub, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".git")))

	dirs := CoreDirs(coreio.Local, sub)
	// Closest first: sub/.core, then repo/.core. Walk stops at repo (.git boundary).
	assert.GreaterOrEqual(t, len(dirs), 2)
	assert.Equal(t, filepath.Join(sub, ".core"), dirs[0])
	assert.Equal(t, filepath.Join(repo, ".core"), dirs[1])
}

func TestDiscover_CoreDirs_Bad(t *testing.T) {
	// A directory tree with no .core anywhere just returns the home layer (if any).
	tmp := t.TempDir()
	dirs := CoreDirs(coreio.Local, tmp)
	for _, dir := range dirs {
		assert.NotContains(t, dir, tmp)
	}
}

func TestDiscover_FindManifest_Good(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	sub := filepath.Join(repo, "service")
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(sub, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".git")))
	// Only the repo-level .core/ has build.yaml.
	assert.NoError(t, coreio.Local.Write(filepath.Join(repo, ".core", "build.yaml"), "name: core\n"))

	path := FindManifest(coreio.Local, sub, FileBuild)
	assert.Equal(t, filepath.Join(repo, ".core", "build.yaml"), path)
}

func TestDiscover_FindManifest_Ugly(t *testing.T) {
	// Missing file returns empty string, not an error.
	tmp := t.TempDir()
	assert.Empty(t, FindManifest(coreio.Local, tmp, FileBuild))
}

func TestDiscover_EnvOverridesDiscovered_Good(t *testing.T) {
	// .core/ convention §5.3: "Env vars override everything."
	// A discovered file value must be shadowed by CORE_CONFIG_* at Get time.
	tmp := t.TempDir()
	t.Setenv("CORE_CONFIG_APP_NAME", "env-wins")

	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(tmp, ".core")))
	assert.NoError(t, coreio.Local.Write(
		filepath.Join(tmp, ".core", "config.yaml"),
		"app:\n  name: fromfile\n",
	))

	cfg, err := DiscoverFrom(tmp, WithMedium(coreio.Local))
	assert.NoError(t, err)

	var name string
	assert.NoError(t, cfg.Get("app.name", &name))
	assert.Equal(t, "env-wins", name)
}

func TestDiscover_MergeFillsGaps_Good(t *testing.T) {
	// Project .core/ wins over global .core/ — global only fills gaps.
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".git")))
	assert.NoError(t, coreio.Local.Write(
		filepath.Join(repo, ".core", "config.yaml"),
		"app:\n  name: project\n",
	))

	cfg, err := DiscoverFrom(repo, WithMedium(coreio.Local))
	assert.NoError(t, err)

	var name string
	assert.NoError(t, cfg.Get("app.name", &name))
	assert.Equal(t, "project", name)
}

func TestDiscover_CommitDoesNotLeakInherited_Good(t *testing.T) {
	// Regression guard: Commit on a discovered Config must only persist the
	// owning file's keys + Set() calls, never inherited layer values — or
	// global ~/.core/ secrets would spray into every project config.
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".git")))
	assert.NoError(t, coreio.Local.Write(
		filepath.Join(repo, ".core", "config.yaml"),
		"secret:\n  token: GLOBAL_ONLY\n",
	))

	commitPath := filepath.Join(tmp, "newcfg.yaml")
	cfg, err := DiscoverFrom(repo, WithMedium(coreio.Local), WithPath(commitPath))
	assert.NoError(t, err)

	// Set our own key then Commit; the inherited GLOBAL_ONLY must NOT appear.
	assert.NoError(t, cfg.Set("dev.shell", "zsh"))
	assert.NoError(t, cfg.Commit())

	body, err := coreio.Local.Read(commitPath)
	assert.NoError(t, err)
	assert.Contains(t, body, "dev:")
	assert.Contains(t, body, "shell: zsh")
	assert.NotContains(t, body, "GLOBAL_ONLY")
	assert.NotContains(t, body, "secret:")
}

func TestDiscover_DiscoverFrom_GlobalFallback_Good(t *testing.T) {
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	home := core.Env("DIR_HOME")

	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(repo, ".git")))
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(home, ".core")))
	assert.NoError(t, coreio.Local.Write(filepath.Join(home, ".core", "config.yaml"), "app:\n  name: global\n"))

	cfg, err := DiscoverFrom(repo, WithMedium(coreio.Local))
	assert.NoError(t, err)

	var name string
	assert.NoError(t, cfg.Get("app.name", &name))
	assert.Equal(t, "global", name)
}
