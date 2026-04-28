package config

import (
	core "dappco.re/go"
	"path/filepath"

	coreio "dappco.re/go/io"
)

func TestWorkspace_FindWorkspaceManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "workspace", "repo")
	child := filepath.Join(root, "service")

	core.AssertNoError(t, m.EnsureDir(filepath.Join(root, ".core")))
	core.AssertNoError(t, m.EnsureDir(child))
	core.AssertNoError(t, m.Write(filepath.Join(root, ".core", FileWorkspace), "version: 1\ndependencies:\n  - core-php\nactive: core-php\npackages_dir: ./packages\n"))

	path := FindWorkspaceManifest(m, child)
	core.AssertEqual(t, filepath.Join(root, ".core", FileWorkspace), path)
}

func TestWorkspace_ResolveWorkspaceManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "workspace", "resolve")

	core.AssertNoError(t, m.EnsureDir(filepath.Join(root, ".core")))
	core.AssertNoError(t, m.Write(filepath.Join(root, ".core", FileWorkspace), "version: 1\ndependencies:\n  - core-php\nactive: core-php\npackages_dir: ./packages\nsettings:\n  suggest_core_commands: true\n"))

	manifest, err := ResolveWorkspaceManifest(m, root)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, manifest)
	core.AssertEqual(t, []string{"core-php"}, manifest.Dependencies)
	core.AssertEqual(t, "core-php", manifest.Active)
	core.AssertEqual(t, "./packages", manifest.PackagesDir)
	core.AssertEqual(t, true, manifest.Settings["suggest_core_commands"])
}

func TestWorkspace_ResolveWorkspaceManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	manifest, err := ResolveWorkspaceManifest(m, filepath.Join("/", "workspace", "missing"))
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no workspace manifest could be detected")
}

func TestWorkspace_ResolveWorkspaceManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "workspace", "ugly")

	core.AssertNoError(t, m.EnsureDir(filepath.Join(root, ".core")))
	core.AssertNoError(t, m.Write(filepath.Join(root, ".core", FileWorkspace), "version: [broken yaml"))

	manifest, err := ResolveWorkspaceManifest(m, root)
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
}

func TestWorkspace_FindWorkspaceRoot_Good(t *core.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "workspace", "root")
	child := filepath.Join(root, "service")

	core.AssertNoError(t, m.EnsureDir(filepath.Join(root, ".core")))
	core.AssertNoError(t, m.EnsureDir(child))
	core.AssertNoError(t, m.Write(filepath.Join(root, ".core", FileWorkspace), "version: 1\n"))

	core.AssertEqual(t, root, FindWorkspaceRoot(m, child))
}

func TestWorkspace_FindWorkspaceRoot_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	start := filepath.Join("/", "workspace", "none")
	got := FindWorkspaceRoot(m, start)
	core.AssertEmpty(t, got)
}

func TestWorkspace_FindWorkspaceRoot_Ugly(t *core.T) {
	m := falseExistsMedium{coreio.NewMockMedium()}
	start := filepath.Join("/", "workspace", "repo", "file.go")
	got := FindWorkspaceRoot(m, start)
	core.AssertEqual(t, "", got)
}
