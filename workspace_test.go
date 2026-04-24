package config

import (
	"path/filepath"
	"testing"

	coreio "dappco.re/go/io"
	"github.com/stretchr/testify/assert"
)

func TestWorkspace_FindWorkspaceManifest_Good(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")
	child := filepath.Join(root, "service")

	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(root, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(child))
	assert.NoError(t, coreio.Local.Write(filepath.Join(root, ".core", FileWorkspace), "version: 1\ndependencies:\n  - core-php\nactive: core-php\npackages_dir: ./packages\n"))

	path := FindWorkspaceManifest(coreio.Local, child)
	assert.Equal(t, filepath.Join(root, ".core", FileWorkspace), path)
}

func TestWorkspace_ResolveWorkspaceManifest_Good(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")

	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(root, ".core")))
	assert.NoError(t, coreio.Local.Write(filepath.Join(root, ".core", FileWorkspace), "version: 1\ndependencies:\n  - core-php\nactive: core-php\npackages_dir: ./packages\nsettings:\n  suggest_core_commands: true\n"))

	manifest, err := ResolveWorkspaceManifest(coreio.Local, root)
	assert.NoError(t, err)
	assert.NotNil(t, manifest)
	assert.Equal(t, []string{"core-php"}, manifest.Dependencies)
	assert.Equal(t, "core-php", manifest.Active)
	assert.Equal(t, "./packages", manifest.PackagesDir)
	assert.Equal(t, true, manifest.Settings["suggest_core_commands"])
}

func TestWorkspace_ResolveWorkspaceManifest_Bad(t *testing.T) {
	tmp := t.TempDir()

	manifest, err := ResolveWorkspaceManifest(coreio.Local, tmp)
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workspace manifest could be detected")
}

func TestWorkspace_ResolveWorkspaceManifest_Ugly(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")

	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(root, ".core")))
	assert.NoError(t, coreio.Local.Write(filepath.Join(root, ".core", FileWorkspace), "version: [broken yaml"))

	manifest, err := ResolveWorkspaceManifest(coreio.Local, root)
	assert.Nil(t, manifest)
	assert.Error(t, err)
}

func TestWorkspace_FindWorkspaceRoot_Good(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")
	child := filepath.Join(root, "service")

	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(root, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(child))
	assert.NoError(t, coreio.Local.Write(filepath.Join(root, ".core", FileWorkspace), "version: 1\n"))

	assert.Equal(t, root, FindWorkspaceRoot(coreio.Local, child))
}

func TestWorkspace_FindWorkspaceRoot_Bad(t *testing.T) {
	tmp := t.TempDir()

	assert.Empty(t, FindWorkspaceRoot(coreio.Local, tmp))
}
