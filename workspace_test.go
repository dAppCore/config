package config

import (
	"path/filepath"
	"testing"

	coreio "dappco.re/go/io"
	"github.com/stretchr/testify/assert"
)

func TestWorkspace_FindWorkspaceManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "workspace", "repo")
	child := filepath.Join(root, "service")

	assert.NoError(t, m.EnsureDir(filepath.Join(root, ".core")))
	assert.NoError(t, m.EnsureDir(child))
	assert.NoError(t, m.Write(filepath.Join(root, ".core", FileWorkspace), "version: 1\ndependencies:\n  - core-php\nactive: core-php\npackages_dir: ./packages\n"))

	path := FindWorkspaceManifest(m, child)
	assert.Equal(t, filepath.Join(root, ".core", FileWorkspace), path)
}

func TestWorkspace_ResolveWorkspaceManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "workspace", "resolve")

	assert.NoError(t, m.EnsureDir(filepath.Join(root, ".core")))
	assert.NoError(t, m.Write(filepath.Join(root, ".core", FileWorkspace), "version: 1\ndependencies:\n  - core-php\nactive: core-php\npackages_dir: ./packages\nsettings:\n  suggest_core_commands: true\n"))

	manifest, err := ResolveWorkspaceManifest(m, root)
	assert.NoError(t, err)
	assert.NotNil(t, manifest)
	assert.Equal(t, []string{"core-php"}, manifest.Dependencies)
	assert.Equal(t, "core-php", manifest.Active)
	assert.Equal(t, "./packages", manifest.PackagesDir)
	assert.Equal(t, true, manifest.Settings["suggest_core_commands"])
}

func TestWorkspace_ResolveWorkspaceManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	manifest, err := ResolveWorkspaceManifest(m, filepath.Join("/", "workspace", "missing"))
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no workspace manifest could be detected")
}

func TestWorkspace_ResolveWorkspaceManifest_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "workspace", "ugly")

	assert.NoError(t, m.EnsureDir(filepath.Join(root, ".core")))
	assert.NoError(t, m.Write(filepath.Join(root, ".core", FileWorkspace), "version: [broken yaml"))

	manifest, err := ResolveWorkspaceManifest(m, root)
	assert.Nil(t, manifest)
	assert.Error(t, err)
}

func TestWorkspace_FindWorkspaceRoot_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "workspace", "root")
	child := filepath.Join(root, "service")

	assert.NoError(t, m.EnsureDir(filepath.Join(root, ".core")))
	assert.NoError(t, m.EnsureDir(child))
	assert.NoError(t, m.Write(filepath.Join(root, ".core", FileWorkspace), "version: 1\n"))

	assert.Equal(t, root, FindWorkspaceRoot(m, child))
}

func TestWorkspace_FindWorkspaceRoot_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	assert.Empty(t, FindWorkspaceRoot(m, filepath.Join("/", "workspace", "none")))
}
