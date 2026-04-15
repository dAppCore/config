package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type falseExistsMedium struct {
	*coreio.MockMedium
}

func (m falseExistsMedium) Exists(string) bool {
	return false
}

func TestResolve_FindConfigManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	base := filepath.Join(tmp, "workspace")
	repo := filepath.Join(base, "repo")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(base, ".core"),
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".git"),
		child,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	globalConfig := filepath.Join(core.Env("DIR_HOME"), ".core", FileConfig)
	projectConfig := filepath.Join(repo, ".core", FileConfig)
	workspaceRepos := filepath.Join(base, ".core", FileRepos)
	assert.NoError(t, m.Write(globalConfig, "app:\n  name: global\n"))
	assert.NoError(t, m.Write(projectConfig, "app:\n  name: project\n"))
	assert.NoError(t, m.Write(workspaceRepos, "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	assert.Equal(t, projectConfig, FindConfigManifest(m, child))
}

func TestResolve_FindConfigManifest_Bad(t *testing.T) {
	m := falseExistsMedium{coreio.NewMockMedium()}

	assert.Empty(t, FindConfigManifest(m, t.TempDir()))
}

func TestResolve_FindConfigManifest_Ugly(t *testing.T) {
	m := falseExistsMedium{coreio.NewMockMedium()}
	start := filepath.Join(t.TempDir(), "missing", "service")

	assert.Empty(t, FindConfigManifest(m, start))
}

func TestResolve_FindProjectManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	base := filepath.Join(tmp, "workspace")
	repo := filepath.Join(base, "repo")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(base, ".core"),
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".git"),
		child,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}
	assert.NoError(t, m.EnsureDir(filepath.Join(base, ".core")))

	files := map[string]string{
		FileBuild:     "name: core\noutput: dist\ntargets:\n  - linux/amd64\n",
		FileRelease:   "version: 1\narchive:\n  format: tar.gz\n  include:\n    - README.md\nchecksums: true\ngithub:\n  draft: false\n  prerelease: false\nchangelog:\n  include:\n    - feat\n",
		FileRun:       "version: 1\nservices:\n  - name: db\n    image: postgres:16\n    port: 5432\ndev:\n  command: php artisan serve\n  port: 8000\n  watch:\n    - app/\n",
		FileView:      "code: photo-browser\nname: Photo Browser\nsign: " + base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)) + "\npermissions:\n  clipboard: true\n",
		FileManifest:  packageManifestFixture(t),
		FileWorkspace: "version: 1\ndependencies:\n  - core-php\nactive: core-php\npackages_dir: ./packages\n",
		FileIDE:       "version: 1\neditor: nvim\n",
	}

	for name, content := range files {
		assert.NoError(t, m.Write(filepath.Join(repo, ".core", name), content))
	}
	assert.NoError(t, m.Write(filepath.Join(base, ".core", FileBuild), "name: external\noutput: ext\n"))
	assert.NoError(t, m.Write(filepath.Join(base, ".core", FileRepos), "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	cases := []struct {
		name string
		path string
		got  string
	}{
		{name: "build", path: filepath.Join(repo, ".core", FileBuild), got: FindBuildManifest(m, child)},
		{name: "release", path: filepath.Join(repo, ".core", FileRelease), got: FindReleaseManifest(m, child)},
		{name: "run", path: filepath.Join(repo, ".core", FileRun), got: FindRunManifest(m, child)},
		{name: "view", path: filepath.Join(repo, ".core", FileView), got: FindViewManifest(m, child)},
		{name: "manifest", path: filepath.Join(repo, ".core", FileManifest), got: FindPackageManifest(m, child)},
		{name: "workspace", path: filepath.Join(repo, ".core", FileWorkspace), got: FindWorkspaceManifest(m, child)},
		{name: "ide", path: filepath.Join(repo, ".core", FileIDE), got: FindIDEManifest(m, child)},
		{name: "repos", path: filepath.Join(base, ".core", FileRepos), got: FindReposManifest(m, child)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.path, tc.got)
		})
	}
}

func TestResolve_FindLinuxKitManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".core", LinuxKitDirectory),
		filepath.Join(repo, ".git"),
		child,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}
	lkPath := filepath.Join(repo, ".core", LinuxKitDirectory, FileLinuxKit)
	assert.NoError(t, m.Write(lkPath, "version: 1\n"))

	assert.Equal(t, lkPath, FindLinuxKitManifest(m, child))
}

func TestResolve_ResolveLinuxKitManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".core", LinuxKitDirectory),
		filepath.Join(repo, ".git"),
		child,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}
	assert.NoError(t, m.Write(filepath.Join(repo, ".core", LinuxKitDirectory, FileLinuxKit), "kernel:\n  image: linuxkit/kernel:6.6.0\n"))

	manifest, err := ResolveLinuxKitManifest(m, child)
	require.NoError(t, err)
	assert.Equal(t, "linuxkit/kernel:6.6.0", manifest["kernel"].(map[string]any)["image"])
}

func TestResolve_ResolveLinuxKitManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	manifest, err := ResolveLinuxKitManifest(m, t.TempDir())
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no linuxkit manifest could be detected")
}

func TestResolve_FindProjectManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	base := filepath.Join(tmp, "workspace")
	repo := filepath.Join(base, "repo")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(base, ".core"),
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".git"),
		child,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	assert.Empty(t, FindBuildManifest(m, child))
	assert.Empty(t, FindReleaseManifest(m, child))
	assert.Empty(t, FindRunManifest(m, child))
	assert.Empty(t, FindViewManifest(m, child))
	assert.Empty(t, FindPackageManifest(m, child))
	assert.Empty(t, FindReposManifest(m, child))
	assert.Empty(t, FindWorkspaceManifest(m, child))
	assert.Empty(t, FindIDEManifest(m, child))
}

func TestResolve_FindProjectManifest_Ugly(t *testing.T) {
	// Nil medium falls back to the local filesystem; with no .core tree the
	// project-local wrappers should still return an empty path instead of panicking.
	// Repos are resolved from the shared workspace root and may legitimately
	// exist on the host machine, so this test does not assert on repos.yaml.
	start := filepath.Join(t.TempDir(), "missing", "service")
	assert.Empty(t, FindBuildManifest(nil, start))
	assert.Empty(t, FindReleaseManifest(nil, start))
	assert.Empty(t, FindRunManifest(nil, start))
	assert.Empty(t, FindViewManifest(nil, start))
	assert.Empty(t, FindPackageManifest(nil, start))
	assert.Empty(t, FindIDEManifest(nil, start))
}

func TestResolve_FindUserPath_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	for _, dir := range []string{
		filepath.Join(home, ".core"),
		filepath.Join(home, ".core", DirectoryImages),
		filepath.Join(home, ".core", DirectorySecrets),
		filepath.Join(home, ".core", DirectoryDaemons),
		filepath.Join(home, ".core", DirectoryWorkspaces),
	} {
		require.NoError(t, m.EnsureDir(dir))
	}

	require.NoError(t, m.Write(filepath.Join(home, ".core", FileAgent), "daemon:\n  enabled: true\n"))
	require.NoError(t, m.Write(filepath.Join(home, ".core", FileZone), "zone:\n  name: alpha\n"))
	require.NoError(t, m.Write(filepath.Join(home, ".core", DirectoryImages, FileImagesManifest), "{\"images\":{}}"))

	assert.Equal(t, filepath.Join(home, ".core", FileAgent), FindUserManifest(m, FileAgent))
	assert.Equal(t, filepath.Join(home, ".core", FileAgent), FindAgentManifest(m))
	assert.Equal(t, filepath.Join(home, ".core", FileAgent), FindUserPath(m, "", FileAgent))
	assert.Equal(t, filepath.Join(home, ".core", FileZone), FindZoneManifest(m))
	assert.Equal(t, filepath.Join(home, ".core", DirectoryImages), FindUserImagesDirectory(m))
	assert.Equal(t, filepath.Join(home, ".core", DirectoryImages, FileImagesManifest), FindUserImagesManifest(m))
	assert.Equal(t, filepath.Join(home, ".core", DirectoryImages, FileImagesManifest), FindUserPath(m, DirectoryImages, "", FileImagesManifest))
	assert.Equal(t, filepath.Join(home, ".core", DirectorySecrets), FindUserSecretsDirectory(m))
	assert.Equal(t, filepath.Join(home, ".core", DirectoryDaemons), FindUserDaemonsDirectory(m))
	assert.Equal(t, filepath.Join(home, ".core", DirectoryWorkspaces), FindUserWorkspacesDirectory(m))
	assert.Equal(t, filepath.Join(home, ".core", DirectoryImages), FindUserDirectory(m, DirectoryImages))
}

func TestResolve_FindUserPath_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	assert.Empty(t, FindUserManifest(m, FileAgent))
	assert.Empty(t, FindUserDirectory(m, DirectoryImages))
	assert.Empty(t, FindUserImagesManifest(m))
	assert.Empty(t, FindUserImagesDirectory(m))
	assert.Empty(t, FindUserSecretsDirectory(m))
	assert.Empty(t, FindUserDaemonsDirectory(m))
	assert.Empty(t, FindUserWorkspacesDirectory(m))
	assert.Empty(t, FindUserPath(m, "..", FileAgent))
	assert.Empty(t, FindUserPath(m, DirectoryImages, "../escape"))
	assert.Empty(t, FindManifest(m, t.TempDir(), "../config.yaml"))
}

func TestResolve_FindUserPath_Ugly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	home := t.TempDir()
	externalCore := filepath.Join(t.TempDir(), "shared-core")

	t.Setenv("DIR_HOME", home)
	require.NoError(t, os.MkdirAll(externalCore, 0755))
	require.NoError(t, os.Symlink(externalCore, filepath.Join(home, ".core")))

	assert.Empty(t, FindUserPath(coreio.Local, FileAgent))
	assert.Empty(t, FindUserManifest(coreio.Local, FileAgent))
}

func TestResolve_ResolveUserManifests_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	require.NoError(t, m.EnsureDir(filepath.Join(home, ".core")))
	require.NoError(t, m.Write(filepath.Join(home, ".core", FileAgent), "daemon:\n  enabled: true\nagents:\n  worker:\n    total: 2\n"))
	require.NoError(t, m.Write(filepath.Join(home, ".core", FileZone), "zone:\n  name: alpha\n  services:\n    vpn:\n      enabled: true\n"))

	agent, err := ResolveAgentManifest(m)
	require.NoError(t, err)
	require.NotNil(t, agent)
	assert.True(t, agent.Daemon.Enabled)
	assert.Equal(t, 2, agent.Agents["worker"].Total)

	zone, err := ResolveZoneManifest(m)
	require.NoError(t, err)
	require.NotNil(t, zone)
	assert.Equal(t, "alpha", zone.Zone.Name)
	assert.True(t, zone.Zone.Services.VPN.Enabled)
}

func TestResolve_ResolveUserManifests_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	agent, err := ResolveAgentManifest(m)
	assert.Nil(t, agent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no agent manifest could be detected")

	zone, err := ResolveZoneManifest(m)
	assert.Nil(t, zone)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no zone manifest could be detected")
}

func TestResolve_ResolveUserManifests_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	require.NoError(t, m.EnsureDir(filepath.Join(home, ".core")))
	require.NoError(t, m.Write(filepath.Join(home, ".core", FileAgent), "daemon:\n  enabled: [broken"))
	require.NoError(t, m.Write(filepath.Join(home, ".core", FileZone), "zone:\n  name: [broken"))

	agent, err := ResolveAgentManifest(m)
	assert.Nil(t, agent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")

	zone, err := ResolveZoneManifest(m)
	assert.Nil(t, zone)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestResolve_FindPHPManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	child := filepath.Join(repo, "service")

	require.NoError(t, m.EnsureDir(filepath.Join(repo, ".core")))
	require.NoError(t, m.EnsureDir(filepath.Join(repo, ".git")))
	require.NoError(t, m.EnsureDir(child))
	require.NoError(t, m.Write(filepath.Join(repo, ".core", FilePHP), "version: 1\n"))
	require.NoError(t, m.EnsureDir(filepath.Join(repo, ".core", LinuxKitDirectory)))

	assert.Equal(t, filepath.Join(repo, ".core", FilePHP), FindPHPManifest(m, child))
	assert.Equal(t, filepath.Join(repo, ".core", LinuxKitDirectory), FindLinuxKitDirectory(m, child))
}

func TestResolve_ResolvePHPManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	php, err := ResolvePHPManifest(m, t.TempDir())
	assert.Nil(t, php)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no php manifest could be detected")
}

func TestResolve_ResolvePHPManifest_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")

	require.NoError(t, m.EnsureDir(filepath.Join(repo, ".core")))
	require.NoError(t, m.Write(filepath.Join(repo, ".core", FilePHP), "version: [broken"))

	php, err := ResolvePHPManifest(m, repo)
	assert.Nil(t, php)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestResolve_ResolvePHPManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")

	require.NoError(t, m.EnsureDir(filepath.Join(repo, ".core")))
	require.NoError(t, m.Write(filepath.Join(repo, ".core", FilePHP), "version: 1\nserver:\n  type: php-fpm\n"))

	php, err := ResolvePHPManifest(m, repo)
	require.NoError(t, err)
	require.NotNil(t, php)
	assert.Equal(t, 1, php.Version)
	assert.Equal(t, "php-fpm", php.Server.Type)
}

func TestResolve_FindReposManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	start := filepath.Join(tmp, "workspace", "repo", "service")
	home := core.Env("DIR_HOME")

	require.NoError(t, m.EnsureDir(start))
	require.NoError(t, m.EnsureDir(filepath.Join(home, "Code", Directory)))
	require.NoError(t, m.Write(filepath.Join(home, "Code", Directory, FileRepos), "version: 1\norg: host-uk\nrepos: []\n"))

	assert.Equal(t, filepath.Join(home, "Code", Directory, FileRepos), FindReposManifest(m, start))
}

func TestResolve_FindWorkspaceRegistryManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	start := filepath.Join(tmp, "workspace", "repo", "service")
	home := core.Env("DIR_HOME")

	require.NoError(t, m.EnsureDir(filepath.Dir(filepath.Join(home, "Code", Directory, FileRepos))))
	require.NoError(t, m.Write(filepath.Join(home, "Code", Directory, FileRepos), "version: 1\norg: host-uk\nrepos: []\n"))

	assert.Equal(t, FindReposManifest(m, start), FindWorkspaceRegistryManifest(m, start))
}

func TestResolve_FindWorkspaceRegistryManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	assert.Empty(t, FindWorkspaceRegistryManifest(m, t.TempDir()))
}

func TestResolve_FindWorkspaceRegistryManifest_Ugly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	m := coreio.NewMockMedium()
	home := t.TempDir()
	externalCore := filepath.Join(t.TempDir(), "shared-core")
	start := filepath.Join(t.TempDir(), "workspace", "repo", "service")

	t.Setenv("DIR_HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "Code"), 0755))
	require.NoError(t, os.MkdirAll(externalCore, 0755))
	require.NoError(t, os.Symlink(externalCore, filepath.Join(home, "Code", Directory)))

	assert.Empty(t, FindWorkspaceRegistryManifest(m, start))
}

func TestResolve_ResolveWorkspaceRegistryManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	start := filepath.Join(tmp, "workspace", "repo", "service")
	home := core.Env("DIR_HOME")

	require.NoError(t, m.EnsureDir(filepath.Join(home, "Code", Directory)))
	require.NoError(t, m.Write(filepath.Join(home, "Code", Directory, FileRepos), "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	repos, err := ResolveWorkspaceRegistryManifest(m, start)
	require.NoError(t, err)
	require.NotNil(t, repos)
	assert.Equal(t, "host-uk", repos.Org)
	assert.Len(t, repos.Repos, 1)
}

func TestResolve_ResolveWorkspaceRegistryManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()

	repos, err := ResolveWorkspaceRegistryManifest(m, t.TempDir())
	assert.Nil(t, repos)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no repos manifest could be detected")
}

func TestResolve_ResolveWorkspaceRegistryManifest_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := filepath.Join(tmp, "workspace", "repo", "service")

	require.NoError(t, m.EnsureDir(filepath.Join(home, "Code", Directory)))
	require.NoError(t, m.Write(filepath.Join(home, "Code", Directory, FileRepos), "version: [broken"))

	repos, err := ResolveWorkspaceRegistryManifest(m, start)
	assert.Nil(t, repos)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse manifest")
}

func TestResolve_ResolveImagesManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	require.NoError(t, m.EnsureDir(filepath.Join(home, ".core", DirectoryImages)))
	require.NoError(t, m.Write(filepath.Join(home, ".core", DirectoryImages, FileImagesManifest), `{"images":{"core-dev":{"version":"1.2.3","downloaded":"2026-04-15T12:00:00Z","source":"github"}}}`))

	manifest, err := ResolveImagesManifest(m)
	require.NoError(t, err)
	require.NotNil(t, manifest)
	require.Len(t, manifest.Images, 1)
	assert.Equal(t, "1.2.3", manifest.Images["core-dev"].Version)
}

func TestResolve_WorkspaceSandboxPath_Good(t *testing.T) {
	home := core.Env("DIR_HOME")
	assert.Equal(t, filepath.Join(home, Directory, WorkspaceDirectory, "repo", "dev"), WorkspaceSandboxRoot("repo", "dev"))
	assert.Equal(t, filepath.Join(home, Directory, WorkspaceDirectory, "repo", "dev", WorkspaceSourceDirectory, "app", "main.go"), WorkspaceSandboxSourcePath("repo", "dev", "app", "main.go"))
	assert.Equal(t, filepath.Join(home, Directory, WorkspaceDirectory, "repo", "dev", WorkspaceMetaDirectory, "status.json"), WorkspaceSandboxMetaPath("repo", "dev", "status.json"))
	assert.Equal(t, filepath.Join(home, Directory, WorkspaceDirectory, "repo", "dev", WorkspaceInstructionsFile), WorkspaceSandboxInstructionsPath("repo", "dev"))
}

func TestResolve_WorkspaceSandboxPath_Ugly(t *testing.T) {
	home := core.Env("DIR_HOME")
	assert.Equal(t, filepath.Join(home, Directory, WorkspaceDirectory, "src"), WorkspaceSandboxPath("", "", "", "src", ""))
	assert.Empty(t, WorkspaceSandboxPath("../repo", "dev"))
	assert.Empty(t, WorkspaceSandboxPath("repo", "../dev"))
	assert.Empty(t, WorkspaceSandboxPath("repo", "dev", "../secret"))
}

func TestResolve_ResolveConfigManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := filepath.Join(tmp, "workspace", "repo", "service")

	for _, dir := range []string{
		filepath.Join(home, ".core"),
		start,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	configPath := filepath.Join(home, ".core", FileConfig)
	assert.NoError(t, m.Write(configPath, "app:\n  name: global\n"))

	cfg, err := ResolveConfigManifest(m, start)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, configPath, cfg.Path())

	var name string
	assert.NoError(t, cfg.Get("app.name", &name))
	assert.Equal(t, "global", name)
}

func TestResolve_ResolveConfigManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	start := filepath.Join(t.TempDir(), "workspace", "repo", "service")

	_, err := ResolveConfigManifest(m, start)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no config manifest could be detected")
}

func TestResolve_ResolveConfigManifest_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := filepath.Join(tmp, "workspace", "repo", "service")

	for _, dir := range []string{
		filepath.Join(home, ".core"),
		start,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	configPath := filepath.Join(home, ".core", FileConfig)
	assert.NoError(t, m.Write(configPath, "app:\n  name: [broken yaml"))

	_, err := ResolveConfigManifest(m, start)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config file")
}

func TestResolve_ResolveProjectManifests_Good(t *testing.T) {
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "workspace", "repo")
	workspace := filepath.Join(tmp, "workspace")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".git"),
		child,
		filepath.Join(workspace, ".core"),
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	buildPath := filepath.Join(repo, ".core", FileBuild)
	releasePath := filepath.Join(repo, ".core", FileRelease)
	runPath := filepath.Join(repo, ".core", FileRun)
	viewPath := filepath.Join(repo, ".core", FileView)
	packagePath := filepath.Join(repo, ".core", FileManifest)
	idePath := filepath.Join(repo, ".core", FileIDE)

	assert.NoError(t, m.Write(buildPath, "name: core\noutput: dist\ntargets:\n  - linux/amd64\n"))
	assert.NoError(t, m.Write(releasePath, "version: 1\narchive:\n  format: tar.gz\n  include:\n    - README.md\nchecksums: true\ngithub:\n  draft: false\n  prerelease: false\nchangelog:\n  include:\n    - feat\n"))
	assert.NoError(t, m.Write(runPath, "version: 1\nservices:\n  - name: db\n    image: postgres:16\n    port: 5432\ndev:\n  command: php artisan serve\n  port: 8000\n  watch:\n    - app/\n"))
	assert.NoError(t, m.Write(viewPath, "code: photo-browser\nname: Photo Browser\nsign: "+base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))+"\npermissions:\n  clipboard: true\n"))
	assert.NoError(t, m.Write(packagePath, packageManifestFixture(t)))
	assert.NoError(t, m.Write(idePath, "version: 1\neditor: nvim\n"))
	assert.NoError(t, m.Write(filepath.Join(workspace, ".core", FileRepos), "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	build, err := ResolveBuildManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, "core", build.Name)
	assert.Equal(t, "dist", build.Output)

	release, err := ResolveReleaseManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, true, release.Checksums)
	assert.Equal(t, "tar.gz", release.Archive.Format)

	run, err := ResolveRunManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, "php artisan serve", run.Dev.Command)
	assert.Len(t, run.Services, 1)

	view, err := ResolveViewManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, "photo-browser", view.Code)
	assert.True(t, view.Permissions.Clipboard)

	pkg, err := ResolvePackageManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, "go-io", pkg.Code)
	assert.Equal(t, "Core I/O", pkg.Name)

	ide, err := ResolveIDEManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, 1, ide.Version)
	assert.Equal(t, "nvim", ide.Editor)

	repos, err := ResolveReposManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, "host-uk", repos.Org)
	assert.Len(t, repos.Repos, 1)
}

func TestResolve_ResolveProjectManifests_Bad(t *testing.T) {
	t.Setenv("DIR_HOME", t.TempDir())
	m := coreio.NewMockMedium()
	start := filepath.Join(t.TempDir(), "workspace", "repo", "service")

	_, err := ResolveBuildManifest(m, start)
	assert.Error(t, err)
	_, err = ResolveReleaseManifest(m, start)
	assert.Error(t, err)
	_, err = ResolveRunManifest(m, start)
	assert.Error(t, err)
	_, err = ResolveViewManifest(m, start)
	assert.Error(t, err)
	_, err = ResolvePackageManifest(m, start)
	assert.Error(t, err)
	_, err = ResolveIDEManifest(m, start)
	assert.Error(t, err)
	_, err = ResolveReposManifest(m, start)
	assert.Error(t, err)
}

func TestResolve_FindReposManifest_FallsBackToWorkspaceRoot_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	start := filepath.Join(t.TempDir(), "workspace", "repo", "service")
	reposPath := filepath.Join(core.Env("DIR_HOME"), "Code", ".core", FileRepos)

	assert.NoError(t, m.EnsureDir(filepath.Dir(reposPath)))
	assert.NoError(t, m.Write(reposPath, "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	assert.Equal(t, reposPath, FindReposManifest(m, start))
}

func TestResolve_ResolveProjectManifests_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "workspace", "repo")
	workspace := filepath.Join(tmp, "workspace")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".git"),
		child,
		filepath.Join(workspace, ".core"),
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	assert.NoError(t, m.Write(filepath.Join(repo, ".core", FileBuild), "name: core\noutput: dist\ntargets:\n  - [broken yaml"))
	assert.NoError(t, m.Write(filepath.Join(repo, ".core", FileRelease), "version: 1\narchive:\n  format: [broken yaml"))
	assert.NoError(t, m.Write(filepath.Join(repo, ".core", FileRun), "version: 1\nservices: [broken yaml"))
	assert.NoError(t, m.Write(filepath.Join(repo, ".core", FileView), "code: photo-browser\nname: Photo Browser\npermissions:\n  clipboard: true\n"))
	assert.NoError(t, m.Write(filepath.Join(repo, ".core", FileManifest), "code: go-io\nname: Core I/O\nsign: not-base64\nsign_key: \"\"\n"))
	assert.NoError(t, m.Write(filepath.Join(workspace, ".core", FileRepos), "version: 1\norg: host-uk\nrepos: [broken yaml"))

	_, err := ResolveBuildManifest(m, child)
	assert.Error(t, err)
	_, err = ResolveReleaseManifest(m, child)
	assert.Error(t, err)
	_, err = ResolveRunManifest(m, child)
	assert.Error(t, err)

	_, err = ResolveViewManifest(m, child)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsigned view manifest rejected")

	_, err = ResolvePackageManifest(m, child)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing package sign_key")

	_, err = ResolveReposManifest(m, child)
	assert.Error(t, err)
}

func packageManifestFixture(t *testing.T) string {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	pkg := &PackageManifest{
		Code:    "go-io",
		Name:    "Core I/O",
		Version: "0.3.0",
		Licence: "EUPL-1.2",
		SignKey: hex.EncodeToString(pub),
	}
	msg, err := packageManifestBytes(pkg)
	assert.NoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	out, err := yaml.Marshal(pkg)
	assert.NoError(t, err)
	return string(out)
}
