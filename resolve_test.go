package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"path/filepath"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"gopkg.in/yaml.v3"
)

type falseExistsMedium struct {
	*coreio.MockMedium
}

func (m falseExistsMedium) Exists(string) bool {
	return false
}

func TestResolve_FindConfigManifest_Good(t *core.T) {
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
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	globalConfig := filepath.Join(core.Env("DIR_HOME"), ".core", FileConfig)
	projectConfig := filepath.Join(repo, ".core", FileConfig)
	workspaceRepos := filepath.Join(base, ".core", FileRepos)
	core.AssertNoError(t, m.Write(globalConfig, "app:\n  name: global\n"))
	core.AssertNoError(t, m.Write(projectConfig, "app:\n  name: project\n"))
	core.AssertNoError(t, m.Write(workspaceRepos, "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	core.AssertEqual(t, projectConfig, FindConfigManifest(m, child))
}

func TestResolve_FindConfigManifest_Bad(t *core.T) {
	m := falseExistsMedium{coreio.NewMockMedium()}
	start := t.TempDir()
	got := FindConfigManifest(m, start)
	core.AssertEmpty(t, got)
}

func TestResolve_FindConfigManifest_Ugly(t *core.T) {
	m := falseExistsMedium{coreio.NewMockMedium()}
	start := filepath.Join(t.TempDir(), "missing", "service")

	core.AssertEmpty(t, FindConfigManifest(m, start))
}

func TestResolve_FindProjectManifest_Good(t *core.T) {
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
		core.AssertNoError(t, m.EnsureDir(dir))
	}
	core.AssertNoError(t, m.EnsureDir(filepath.Join(base, ".core")))

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
		core.AssertNoError(t, m.Write(filepath.Join(repo, ".core", name), content))
	}
	core.AssertNoError(t, m.Write(filepath.Join(base, ".core", FileBuild), "name: external\noutput: ext\n"))
	core.AssertNoError(t, m.Write(filepath.Join(base, ".core", FileRepos), "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

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
		t.Run(tc.name, func(t *core.T) {
			core.AssertEqual(t, tc.path, tc.got)
		})
	}
}

func TestResolve_FindLinuxKitManifest_Good(t *core.T) {
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
		core.AssertNoError(t, m.EnsureDir(dir))
	}
	lkPath := filepath.Join(repo, ".core", LinuxKitDirectory, FileLinuxKit)
	core.AssertNoError(t, m.Write(lkPath, "version: 1\n"))

	core.AssertEqual(t, lkPath, FindLinuxKitManifest(m, child))
}

func TestResolve_ResolveLinuxKitManifest_Good(t *core.T) {
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
		core.AssertNoError(t, m.EnsureDir(dir))
	}
	core.AssertNoError(t, m.Write(filepath.Join(repo, ".core", LinuxKitDirectory, FileLinuxKit), "kernel:\n  image: linuxkit/kernel:6.6.0\n"))

	manifest, err := ResolveLinuxKitManifest(m, child)
	core.RequireNoError(t, err)
	core.AssertEqual(t, "linuxkit/kernel:6.6.0", manifest["kernel"].(map[string]any)["image"])
}

func TestResolve_ResolveLinuxKitManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	manifest, err := ResolveLinuxKitManifest(m, t.TempDir())
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no linuxkit manifest could be detected")
}

func TestResolve_FindProjectManifest_Bad(t *core.T) {
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
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	core.AssertEmpty(t, FindBuildManifest(m, child))
	core.AssertEmpty(t, FindReleaseManifest(m, child))
	core.AssertEmpty(t, FindRunManifest(m, child))
	core.AssertEmpty(t, FindViewManifest(m, child))
	core.AssertEmpty(t, FindPackageManifest(m, child))
	core.AssertEmpty(t, FindReposManifest(m, child))
	core.AssertEmpty(t, FindWorkspaceManifest(m, child))
	core.AssertEmpty(t, FindIDEManifest(m, child))
}

func TestResolve_FindProjectManifest_Ugly(t *core.T) {
	// Nil medium falls back to the local filesystem; with no .core tree the
	// project-local wrappers should still return an empty path instead of panicking.
	// Repos are resolved from the shared workspace root and may legitimately
	// exist on the host machine, so this test does not assert on repos.yaml.
	start := filepath.Join(t.TempDir(), "missing", "service")
	core.AssertEmpty(t, FindBuildManifest(nil, start))
	core.AssertEmpty(t, FindReleaseManifest(nil, start))
	core.AssertEmpty(t, FindRunManifest(nil, start))
	core.AssertEmpty(t, FindViewManifest(nil, start))
	core.AssertEmpty(t, FindPackageManifest(nil, start))
	core.AssertEmpty(t, FindIDEManifest(nil, start))
}

func TestResolve_FindUserPath_Good(t *core.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	for _, dir := range []string{
		filepath.Join(home, ".core"),
		filepath.Join(home, ".core", DirectoryImages),
		filepath.Join(home, ".core", DirectorySecrets),
		filepath.Join(home, ".core", DirectoryDaemons),
		filepath.Join(home, ".core", DirectoryWorkspaces),
	} {
		core.RequireNoError(t, m.EnsureDir(dir))
	}

	core.RequireNoError(t, m.Write(filepath.Join(home, ".core", FileAgent), "daemon:\n  enabled: true\n"))
	core.RequireNoError(t, m.Write(filepath.Join(home, ".core", FileZone), "zone:\n  name: alpha\n  identity: '@alpha@lthn'\n"))
	core.RequireNoError(t, m.Write(filepath.Join(home, ".core", DirectoryImages, FileImagesManifest), "{\"images\":{}}"))

	core.AssertEqual(t, filepath.Join(home, ".core", FileAgent), FindUserManifest(m, FileAgent))
	core.AssertEqual(t, filepath.Join(home, ".core", FileAgent), FindAgentManifest(m))
	core.AssertEqual(t, filepath.Join(home, ".core", FileAgent), FindUserPath(m, "", FileAgent))
	core.AssertEqual(t, filepath.Join(home, ".core", FileZone), FindZoneManifest(m))
	core.AssertEqual(t, filepath.Join(home, ".core", DirectoryImages), FindUserImagesDirectory(m))
	core.AssertEqual(t, filepath.Join(home, ".core", DirectoryImages, FileImagesManifest), FindUserImagesManifest(m))
	core.AssertEqual(t, filepath.Join(home, ".core", DirectoryImages, FileImagesManifest), FindUserPath(m, DirectoryImages, "", FileImagesManifest))
	core.AssertEqual(t, filepath.Join(home, ".core", DirectorySecrets), FindUserSecretsDirectory(m))
	core.AssertEqual(t, filepath.Join(home, ".core", DirectoryDaemons), FindUserDaemonsDirectory(m))
	core.AssertEqual(t, filepath.Join(home, ".core", DirectoryWorkspaces), FindUserWorkspacesDirectory(m))
	core.AssertEqual(t, filepath.Join(home, ".core", DirectoryImages), FindUserDirectory(m, DirectoryImages))
}

func TestResolve_FindUserPath_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	core.AssertEmpty(t, FindUserManifest(m, FileAgent))
	core.AssertEmpty(t, FindUserDirectory(m, DirectoryImages))
	core.AssertEmpty(t, FindUserImagesManifest(m))
	core.AssertEmpty(t, FindUserImagesDirectory(m))
	core.AssertEmpty(t, FindUserSecretsDirectory(m))
	core.AssertEmpty(t, FindUserDaemonsDirectory(m))
	core.AssertEmpty(t, FindUserWorkspacesDirectory(m))
	core.AssertEmpty(t, FindUserPath(m, "..", FileAgent))
	core.AssertEmpty(t, FindUserPath(m, DirectoryImages, "../escape"))
	core.AssertEmpty(t, FindManifest(m, t.TempDir(), "../config.yaml"))
}

func TestResolve_FindUserPath_Ugly(t *core.T) {
	home := core.Env("DIR_HOME")
	coreDir := filepath.Join(home, ".core")
	m := symlinkMockMedium{
		MockMedium: coreio.NewMockMedium(),
		symlinks:   map[string]bool{coreDir: true},
	}
	core.RequireNoError(t, m.EnsureDir(coreDir))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileAgent), "daemon:\n  enabled: true\n"))

	core.AssertEmpty(t, FindUserPath(m, FileAgent))
	core.AssertEmpty(t, FindUserManifest(m, FileAgent))
}

func TestResolve_ResolveUserManifests_Good(t *core.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(filepath.Join(home, ".core")))
	core.RequireNoError(t, m.Write(filepath.Join(home, ".core", FileAgent), "daemon:\n  enabled: true\nagents:\n  worker:\n    total: 2\n"))
	core.RequireNoError(t, m.Write(filepath.Join(home, ".core", FileZone), "zone:\n  name: alpha\n  identity: '@alpha@lthn'\n  services:\n    vpn:\n      enabled: true\n"))

	agent, err := ResolveAgentManifest(m)
	core.RequireNoError(t, err)
	core.RequireTrue(t, agent != nil)
	core.AssertTrue(t, agent.Daemon.Enabled)
	core.AssertEqual(t, 2, agent.Agents["worker"].Total)

	zone, err := ResolveZoneManifest(m)
	core.RequireNoError(t, err)
	core.RequireTrue(t, zone != nil)
	core.AssertEqual(t, "alpha", zone.Zone.Name)
	core.AssertTrue(t, zone.Zone.Services.VPN.Enabled)
}

func TestResolve_ResolveUserManifests_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	agent, err := ResolveAgentManifest(m)
	core.AssertNil(t, agent)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no agent manifest could be detected")

	zone, err := ResolveZoneManifest(m)
	core.AssertNil(t, zone)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no zone manifest could be detected")
}

func TestResolve_ResolveUserManifests_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(filepath.Join(home, ".core")))
	core.RequireNoError(t, m.Write(filepath.Join(home, ".core", FileAgent), "daemon:\n  enabled: [broken"))
	core.RequireNoError(t, m.Write(filepath.Join(home, ".core", FileZone), "zone:\n  name: [broken"))

	agent, err := ResolveAgentManifest(m)
	core.AssertNil(t, agent)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to parse manifest")

	zone, err := ResolveZoneManifest(m)
	core.AssertNil(t, zone)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to parse manifest")
}

func TestResolve_FindPHPManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")
	child := filepath.Join(repo, "service")

	core.RequireNoError(t, m.EnsureDir(filepath.Join(repo, ".core")))
	core.RequireNoError(t, m.EnsureDir(filepath.Join(repo, ".git")))
	core.RequireNoError(t, m.EnsureDir(child))
	core.RequireNoError(t, m.Write(filepath.Join(repo, ".core", FilePHP), "version: 1\n"))
	core.RequireNoError(t, m.EnsureDir(filepath.Join(repo, ".core", LinuxKitDirectory)))

	core.AssertEqual(t, filepath.Join(repo, ".core", FilePHP), FindPHPManifest(m, child))
	core.AssertEqual(t, filepath.Join(repo, ".core", LinuxKitDirectory), FindLinuxKitDirectory(m, child))
}

func TestResolve_ResolvePHPManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	php, err := ResolvePHPManifest(m, t.TempDir())
	core.AssertNil(t, php)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no php manifest could be detected")
}

func TestResolve_ResolvePHPManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")

	core.RequireNoError(t, m.EnsureDir(filepath.Join(repo, ".core")))
	core.RequireNoError(t, m.Write(filepath.Join(repo, ".core", FilePHP), "version: [broken"))

	php, err := ResolvePHPManifest(m, repo)
	core.AssertNil(t, php)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to parse manifest")
}

func TestResolve_ResolvePHPManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "repo")

	core.RequireNoError(t, m.EnsureDir(filepath.Join(repo, ".core")))
	core.RequireNoError(t, m.Write(filepath.Join(repo, ".core", FilePHP), "version: 1\nserver:\n  type: php-fpm\n"))

	php, err := ResolvePHPManifest(m, repo)
	core.RequireNoError(t, err)
	core.RequireTrue(t, php != nil)
	core.AssertEqual(t, 1, php.Version)
	core.AssertEqual(t, "php-fpm", php.Server.Type)
}

func TestResolve_FindReposManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	start := filepath.Join(tmp, "workspace", "repo", "service")
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(start))
	core.RequireNoError(t, m.EnsureDir(filepath.Join(home, "Code", Directory)))
	core.RequireNoError(t, m.Write(filepath.Join(home, "Code", Directory, FileRepos), "version: 1\norg: host-uk\nrepos: []\n"))

	core.AssertEqual(t, filepath.Join(home, "Code", Directory, FileRepos), FindReposManifest(m, start))
}

func TestResolve_FindWorkspaceRegistryManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	start := filepath.Join(tmp, "workspace", "repo", "service")
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(filepath.Dir(filepath.Join(home, "Code", Directory, FileRepos))))
	core.RequireNoError(t, m.Write(filepath.Join(home, "Code", Directory, FileRepos), "version: 1\norg: host-uk\nrepos: []\n"))

	core.AssertEqual(t, FindReposManifest(m, start), FindWorkspaceRegistryManifest(m, start))
}

func TestResolve_FindWorkspaceRegistryManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	start := t.TempDir()
	got := FindWorkspaceRegistryManifest(m, start)
	core.AssertEmpty(t, got)
}

func TestResolve_FindWorkspaceRegistryManifest_Ugly(t *core.T) {
	home := core.Env("DIR_HOME")
	coreDir := filepath.Join(home, "Code", Directory)
	start := filepath.Join("workspace", "repo", "service")
	m := symlinkMockMedium{
		MockMedium: coreio.NewMockMedium(),
		symlinks:   map[string]bool{coreDir: true},
	}
	core.RequireNoError(t, m.EnsureDir(coreDir))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileRepos), "version: 1\norg: host-uk\nrepos: []\n"))

	core.AssertEmpty(t, FindWorkspaceRegistryManifest(m, start))
}

func TestResolve_ResolveWorkspaceRegistryManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	start := filepath.Join(tmp, "workspace", "repo", "service")
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(filepath.Join(home, "Code", Directory)))
	core.RequireNoError(t, m.Write(filepath.Join(home, "Code", Directory, FileRepos), "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	repos, err := ResolveWorkspaceRegistryManifest(m, start)
	core.RequireNoError(t, err)
	core.RequireTrue(t, repos != nil)
	core.AssertEqual(t, "host-uk", repos.Org)
	core.AssertLen(t, repos.Repos, 1)
}

func TestResolve_ResolveWorkspaceRegistryManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	repos, err := ResolveWorkspaceRegistryManifest(m, t.TempDir())
	core.AssertNil(t, repos)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no repos manifest could be detected")
}

func TestResolve_ResolveWorkspaceRegistryManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := filepath.Join(tmp, "workspace", "repo", "service")

	core.RequireNoError(t, m.EnsureDir(filepath.Join(home, "Code", Directory)))
	core.RequireNoError(t, m.Write(filepath.Join(home, "Code", Directory, FileRepos), "version: [broken"))

	repos, err := ResolveWorkspaceRegistryManifest(m, start)
	core.AssertNil(t, repos)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to parse manifest")
}

func TestResolve_ResolveImagesManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(filepath.Join(home, ".core", DirectoryImages)))
	core.RequireNoError(t, m.Write(filepath.Join(home, ".core", DirectoryImages, FileImagesManifest), `{"images":{"core-dev":{"version":"1.2.3","downloaded":"2026-04-15T12:00:00Z","source":"github"}}}`))

	manifest, err := ResolveImagesManifest(m)
	core.RequireNoError(t, err)
	core.RequireTrue(t, manifest != nil)
	core.AssertLen(t, manifest.Images, 1)
	core.AssertEqual(t, "1.2.3", manifest.Images["core-dev"].Version)
}

func TestResolve_WorkspaceSandboxPath_Good(t *core.T) {
	home := core.Env("DIR_HOME")
	core.AssertEqual(t, filepath.Join(home, Directory, WorkspaceDirectory, "repo", "dev"), WorkspaceSandboxRoot("repo", "dev"))
	core.AssertEqual(t, filepath.Join(home, Directory, WorkspaceDirectory, "repo", "dev", WorkspaceSourceDirectory, "app", "main.go"), WorkspaceSandboxSourcePath("repo", "dev", "app", "main.go"))
	core.AssertEqual(t, filepath.Join(home, Directory, WorkspaceDirectory, "repo", "dev", WorkspaceMetaDirectory, "status.json"), WorkspaceSandboxMetaPath("repo", "dev", "status.json"))
	core.AssertEqual(t, filepath.Join(home, Directory, WorkspaceDirectory, "repo", "dev", WorkspaceInstructionsFile), WorkspaceSandboxInstructionsPath("repo", "dev"))
}

func TestResolve_WorkspaceSandboxPath_Ugly(t *core.T) {
	home := core.Env("DIR_HOME")
	core.AssertEqual(t, filepath.Join(home, Directory, WorkspaceDirectory, "src"), WorkspaceSandboxPath("", "", "", "src", ""))
	core.AssertEmpty(t, WorkspaceSandboxPath("../repo", "dev"))
	core.AssertEmpty(t, WorkspaceSandboxPath("repo", "../dev"))
	core.AssertEmpty(t, WorkspaceSandboxPath("repo", "dev", "../secret"))
}

func TestResolve_ResolveConfigManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := filepath.Join(tmp, "workspace", "repo", "service")

	for _, dir := range []string{
		filepath.Join(home, ".core"),
		start,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	configPath := filepath.Join(home, ".core", FileConfig)
	core.AssertNoError(t, m.Write(configPath, "app:\n  name: global\n"))

	cfg, err := ResolveConfigManifest(m, start)
	core.AssertNoError(t, err)
	core.AssertNotNil(t, cfg)
	core.AssertEqual(t, configPath, cfg.Path())

	var name string
	core.AssertNoError(t, cfg.Get("app.name", &name))
	core.AssertEqual(t, "global", name)
}

func TestResolve_ResolveConfigManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	start := filepath.Join(t.TempDir(), "workspace", "repo", "service")

	_, err := ResolveConfigManifest(m, start)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no config manifest could be detected")
}

func TestResolve_ResolveConfigManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := filepath.Join(tmp, "workspace", "repo", "service")

	for _, dir := range []string{
		filepath.Join(home, ".core"),
		start,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	configPath := filepath.Join(home, ".core", FileConfig)
	core.AssertNoError(t, m.Write(configPath, "app:\n  name: [broken yaml"))

	_, err := ResolveConfigManifest(m, start)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to load config file")
}

func TestResolve_ResolveProjectManifests_Good(t *core.T) {
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
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	buildPath := filepath.Join(repo, ".core", FileBuild)
	releasePath := filepath.Join(repo, ".core", FileRelease)
	runPath := filepath.Join(repo, ".core", FileRun)
	viewPath := filepath.Join(repo, ".core", FileView)
	packagePath := filepath.Join(repo, ".core", FileManifest)
	idePath := filepath.Join(repo, ".core", FileIDE)

	core.AssertNoError(t, m.Write(buildPath, "name: core\noutput: dist\ntargets:\n  - linux/amd64\n"))
	core.AssertNoError(t, m.Write(releasePath, "version: 1\narchive:\n  format: tar.gz\n  include:\n    - README.md\nchecksums: true\ngithub:\n  draft: false\n  prerelease: false\nchangelog:\n  include:\n    - feat\n"))
	core.AssertNoError(t, m.Write(runPath, "version: 1\nservices:\n  - name: db\n    image: postgres:16\n    port: 5432\ndev:\n  command: php artisan serve\n  port: 8000\n  watch:\n    - app/\n"))
	core.AssertNoError(t, m.Write(viewPath, "code: photo-browser\nname: Photo Browser\nsign: "+base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))+"\npermissions:\n  clipboard: true\n"))
	core.AssertNoError(t, m.Write(packagePath, packageManifestFixture(t)))
	core.AssertNoError(t, m.Write(idePath, "version: 1\neditor: nvim\n"))
	core.AssertNoError(t, m.Write(filepath.Join(workspace, ".core", FileRepos), "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	build, err := ResolveBuildManifest(m, child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", build.Name)
	core.AssertEqual(t, "dist", build.Output)

	release, err := ResolveReleaseManifest(m, child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, true, release.Checksums)
	core.AssertEqual(t, "tar.gz", release.Archive.Format)

	run, err := ResolveRunManifest(m, child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "php artisan serve", run.Dev.Command)
	core.AssertLen(t, run.Services, 1)

	view, err := ResolveViewManifest(m, child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "photo-browser", view.Code)
	core.AssertTrue(t, view.Permissions.Clipboard)

	pkg, err := ResolvePackageManifest(m, child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "go-io", pkg.Code)
	core.AssertEqual(t, "Core I/O", pkg.Name)

	ide, err := ResolveIDEManifest(m, child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, ide.Version)
	core.AssertEqual(t, "nvim", ide.Editor)

	repos, err := ResolveReposManifest(m, child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "host-uk", repos.Org)
	core.AssertLen(t, repos.Repos, 1)
}

func TestResolve_ResolveProjectManifests_Bad(t *core.T) {
	t.Setenv("DIR_HOME", t.TempDir())
	m := coreio.NewMockMedium()
	start := filepath.Join(t.TempDir(), "workspace", "repo", "service")

	_, err := ResolveBuildManifest(m, start)
	core.AssertError(t, err)
	_, err = ResolveReleaseManifest(m, start)
	core.AssertError(t, err)
	_, err = ResolveRunManifest(m, start)
	core.AssertError(t, err)
	_, err = ResolveViewManifest(m, start)
	core.AssertError(t, err)
	_, err = ResolvePackageManifest(m, start)
	core.AssertError(t, err)
	_, err = ResolveIDEManifest(m, start)
	core.AssertError(t, err)
	_, err = ResolveReposManifest(m, start)
	core.AssertError(t, err)
}

func TestResolve_FindReposManifest_FallsBackToWorkspaceRoot_Good(t *core.T) {
	m := coreio.NewMockMedium()
	start := filepath.Join(t.TempDir(), "workspace", "repo", "service")
	reposPath := filepath.Join(core.Env("DIR_HOME"), "Code", ".core", FileRepos)

	core.AssertNoError(t, m.EnsureDir(filepath.Dir(reposPath)))
	core.AssertNoError(t, m.Write(reposPath, "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	core.AssertEqual(t, reposPath, FindReposManifest(m, start))
}

func TestResolve_ResolveProjectManifests_Ugly(t *core.T) {
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
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	core.AssertNoError(t, m.Write(filepath.Join(repo, ".core", FileBuild), "name: core\noutput: dist\ntargets:\n  - [broken yaml"))
	core.AssertNoError(t, m.Write(filepath.Join(repo, ".core", FileRelease), "version: 1\narchive:\n  format: [broken yaml"))
	core.AssertNoError(t, m.Write(filepath.Join(repo, ".core", FileRun), "version: 1\nservices: [broken yaml"))
	core.AssertNoError(t, m.Write(filepath.Join(repo, ".core", FileView), "code: photo-browser\nname: Photo Browser\npermissions:\n  clipboard: true\n"))
	core.AssertNoError(t, m.Write(filepath.Join(repo, ".core", FileManifest), "code: go-io\nname: Core I/O\nsign: not-base64\nsign_key: \"\"\n"))
	core.AssertNoError(t, m.Write(filepath.Join(workspace, ".core", FileRepos), "version: 1\norg: host-uk\nrepos: [broken yaml"))

	_, err := ResolveBuildManifest(m, child)
	core.AssertError(t, err)
	_, err = ResolveReleaseManifest(m, child)
	core.AssertError(t, err)
	_, err = ResolveRunManifest(m, child)
	core.AssertError(t, err)

	_, err = ResolveViewManifest(m, child)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsigned view manifest rejected")

	_, err = ResolvePackageManifest(m, child)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "missing package sign_key")

	_, err = ResolveReposManifest(m, child)
	core.AssertError(t, err)
}

func packageManifestFixture(t *core.T) string {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)

	pkg := &PackageManifest{
		Code:    "go-io",
		Name:    "Core I/O",
		Version: "0.3.0",
		Licence: "EUPL-1.2",
		SignKey: hex.EncodeToString(pub),
	}
	msg, err := packageManifestBytes(pkg)
	core.AssertNoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	out, err := yaml.Marshal(pkg)
	core.AssertNoError(t, err)
	return string(out)
}

type axResolveProject struct {
	medium *coreio.MockMedium
	root   string
	child  string
	core   string
}

func axResolveProjectFixture(t *core.T) axResolveProject {
	t.Helper()
	m := coreio.NewMockMedium()
	root := filepath.Join(t.TempDir(), "repo")
	child := filepath.Join(root, "service")
	coreDir := filepath.Join(root, Directory)
	core.RequireNoError(t, m.EnsureDir(coreDir))
	core.RequireNoError(t, m.EnsureDir(filepath.Join(root, ".git")))
	core.RequireNoError(t, m.EnsureDir(child))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileBuild), "version: 1\nproject:\n  name: core\nbuild:\n  flags: [-trimpath]\ntargets:\n  - linux/amd64\n"))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileTest), "version: 1\ncommands:\n  - name: unit\n    run: go test ./...\n"))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileRelease), "version: 1\narchive:\n  format: tar.gz\n  include: [README.md]\nchecksums: true\n"))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileRun), "version: 1\nservices:\n  - name: db\n    image: postgres:16\n    port: 5432\n"))
	view, _ := axSignedView(t)
	viewBody, err := yaml.Marshal(&view)
	core.RequireNoError(t, err)
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileView), string(viewBody)))
	pkg, pub := axSignedPackage(t)
	setManifestTrustKeys(t, hex.EncodeToString(pub))
	pkgBody, err := yaml.Marshal(&pkg)
	core.RequireNoError(t, err)
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileManifest), string(pkgBody)))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileWorkspace), "version: 1\ndependencies: [core/go]\n"))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileIDE), "version: 1\neditor: codex\n"))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FilePHP), "version: 1\nserver:\n  type: php-fpm\n"))
	core.RequireNoError(t, m.EnsureDir(filepath.Join(coreDir, LinuxKitDirectory)))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, LinuxKitDirectory, FileLinuxKit), "kernel:\n  image: linuxkit/kernel:6.6.0\n"))
	return axResolveProject{medium: m, root: root, child: child, core: coreDir}
}

func axResolveUserFixture(t *core.T) (*coreio.MockMedium, string) {
	t.Helper()
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")
	coreDir := filepath.Join(home, Directory)
	core.RequireNoError(t, m.EnsureDir(coreDir))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileAgent), "daemon:\n  enabled: true\nagents:\n  codex:\n    total: 1\n"))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileZone), "zone:\n  name: alpha\n  identity: '@alpha@lthn'\n  services:\n    vpn:\n      enabled: true\n"))
	for _, dir := range []string{DirectoryImages, DirectorySecrets, DirectoryDaemons, DirectoryWorkspaces} {
		core.RequireNoError(t, m.EnsureDir(filepath.Join(coreDir, dir)))
	}
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, DirectoryImages, FileImagesManifest), `{"images":{}}`))
	return m, home
}

func TestResolve_FindUserManifest_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserManifest(m, FileAgent)
	core.AssertEqual(t, filepath.Join(home, Directory, FileAgent), got)
}

func TestResolve_FindUserManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserManifest(m, FileAgent)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{filepath.Join(home, Directory): true}}
	got := FindUserManifest(marked, FileAgent)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserDirectory_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserDirectory(m, DirectoryImages)
	core.AssertEqual(t, filepath.Join(home, Directory, DirectoryImages), got)
}

func TestResolve_FindUserDirectory_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserDirectory(m, DirectoryImages)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserDirectory_Ugly(t *core.T) {
	m, _ := axResolveUserFixture(t)
	got := FindUserDirectory(m, "../images")
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserImagesManifest_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserImagesManifest(m)
	core.AssertEqual(t, filepath.Join(home, Directory, DirectoryImages, FileImagesManifest), got)
}

func TestResolve_FindUserImagesManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserImagesManifest(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserImagesManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{filepath.Join(home, Directory): true}}
	got := FindUserImagesManifest(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserImagesDirectory_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserImagesDirectory(m)
	core.AssertEqual(t, filepath.Join(home, Directory, DirectoryImages), got)
}

func TestResolve_FindUserImagesDirectory_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserImagesDirectory(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserImagesDirectory_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{filepath.Join(home, Directory): true}}
	got := FindUserImagesDirectory(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserSecretsDirectory_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserSecretsDirectory(m)
	core.AssertEqual(t, filepath.Join(home, Directory, DirectorySecrets), got)
}

func TestResolve_FindUserSecretsDirectory_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserSecretsDirectory(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserSecretsDirectory_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{filepath.Join(home, Directory): true}}
	got := FindUserSecretsDirectory(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserDaemonsDirectory_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserDaemonsDirectory(m)
	core.AssertEqual(t, filepath.Join(home, Directory, DirectoryDaemons), got)
}

func TestResolve_FindUserDaemonsDirectory_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserDaemonsDirectory(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserDaemonsDirectory_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{filepath.Join(home, Directory): true}}
	got := FindUserDaemonsDirectory(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserWorkspacesDirectory_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserWorkspacesDirectory(m)
	core.AssertEqual(t, filepath.Join(home, Directory, DirectoryWorkspaces), got)
}

func TestResolve_FindUserWorkspacesDirectory_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserWorkspacesDirectory(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserWorkspacesDirectory_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{filepath.Join(home, Directory): true}}
	got := FindUserWorkspacesDirectory(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindBuildManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindBuildManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, filepath.Join(fixture.core, FileBuild), got)
}

func TestResolve_FindBuildManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindBuildManifest(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindBuildManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindBuildManifest(marked, fixture.child)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveBuildManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got, err := ResolveBuildManifest(fixture.medium, fixture.child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", got.Project.Name)
}

func TestResolve_ResolveBuildManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := ResolveBuildManifest(m, t.TempDir())
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveBuildManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(filepath.Join(fixture.core, FileBuild), "targets:\n  - invalid-target\n"))
	got, err := ResolveBuildManifest(fixture.medium, fixture.child)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindTestManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindTestManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, filepath.Join(fixture.core, FileTest), got)
}

func TestResolve_FindTestManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindTestManifest(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindTestManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindTestManifest(marked, fixture.child)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindReleaseManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindReleaseManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, filepath.Join(fixture.core, FileRelease), got)
}

func TestResolve_FindReleaseManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindReleaseManifest(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindReleaseManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindReleaseManifest(marked, fixture.child)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveReleaseManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got, err := ResolveReleaseManifest(fixture.medium, fixture.child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "tar.gz", got.Archive.Format)
}

func TestResolve_ResolveReleaseManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := ResolveReleaseManifest(m, t.TempDir())
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveReleaseManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(filepath.Join(fixture.core, FileRelease), "version: [broken"))
	got, err := ResolveReleaseManifest(fixture.medium, fixture.child)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindRunManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindRunManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, filepath.Join(fixture.core, FileRun), got)
}

func TestResolve_FindRunManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindRunManifest(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindRunManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindRunManifest(marked, fixture.child)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveRunManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got, err := ResolveRunManifest(fixture.medium, fixture.child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "db", got.Services[0].Name)
}

func TestResolve_ResolveRunManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := ResolveRunManifest(m, t.TempDir())
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveRunManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(filepath.Join(fixture.core, FileRun), "services: [broken"))
	got, err := ResolveRunManifest(fixture.medium, fixture.child)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindViewManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindViewManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, filepath.Join(fixture.core, FileView), got)
}

func TestResolve_FindViewManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindViewManifest(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindViewManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindViewManifest(marked, fixture.child)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveViewManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got, err := ResolveViewManifest(fixture.medium, fixture.child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "photo-browser", got.Code)
}

func TestResolve_ResolveViewManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := ResolveViewManifest(m, t.TempDir())
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveViewManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(filepath.Join(fixture.core, FileView), "code: ax\nname: AX\n"))
	got, err := ResolveViewManifest(fixture.medium, fixture.child)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindPackageManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindPackageManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, filepath.Join(fixture.core, FileManifest), got)
}

func TestResolve_FindPackageManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindPackageManifest(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindPackageManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindPackageManifest(marked, fixture.child)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolvePackageManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got, err := ResolvePackageManifest(fixture.medium, fixture.child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "go-config", got.Code)
}

func TestResolve_ResolvePackageManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := ResolvePackageManifest(m, t.TempDir())
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolvePackageManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(filepath.Join(fixture.core, FileManifest), "code: ax\nname: AX\nsign: not-base64\nsign_key: \"\"\n"))
	got, err := ResolvePackageManifest(fixture.medium, fixture.child)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindAgentManifest_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindAgentManifest(m)
	core.AssertEqual(t, filepath.Join(home, Directory, FileAgent), got)
}

func TestResolve_FindAgentManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindAgentManifest(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindAgentManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{filepath.Join(home, Directory): true}}
	got := FindAgentManifest(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveAgentManifest_Good(t *core.T) {
	m, _ := axResolveUserFixture(t)
	got, err := ResolveAgentManifest(m)
	core.AssertNoError(t, err)
	core.AssertTrue(t, got.Daemon.Enabled)
}

func TestResolve_ResolveAgentManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := ResolveAgentManifest(m)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveAgentManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	core.RequireNoError(t, m.Write(filepath.Join(home, Directory, FileAgent), "daemon: [broken"))
	got, err := ResolveAgentManifest(m)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindZoneManifest_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindZoneManifest(m)
	core.AssertEqual(t, filepath.Join(home, Directory, FileZone), got)
}

func TestResolve_FindZoneManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindZoneManifest(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindZoneManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{filepath.Join(home, Directory): true}}
	got := FindZoneManifest(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveZoneManifest_Good(t *core.T) {
	m, _ := axResolveUserFixture(t)
	got, err := ResolveZoneManifest(m)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "alpha", got.Zone.Name)
}

func TestResolve_ResolveZoneManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := ResolveZoneManifest(m)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveZoneManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	core.RequireNoError(t, m.Write(filepath.Join(home, Directory, FileZone), "zone: [broken"))
	got, err := ResolveZoneManifest(m)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindWorkspaceManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindWorkspaceManifest(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindWorkspaceManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindWorkspaceManifest(marked, fixture.child)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindIDEManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindIDEManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, filepath.Join(fixture.core, FileIDE), got)
}

func TestResolve_FindIDEManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindIDEManifest(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindIDEManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindIDEManifest(marked, fixture.child)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveIDEManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got, err := ResolveIDEManifest(fixture.medium, fixture.child)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "codex", got.Editor)
}

func TestResolve_ResolveIDEManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := ResolveIDEManifest(m, t.TempDir())
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveIDEManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(filepath.Join(fixture.core, FileIDE), "version: [broken"))
	got, err := ResolveIDEManifest(fixture.medium, fixture.child)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindLinuxKitDirectory_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindLinuxKitDirectory(fixture.medium, fixture.child)
	core.AssertEqual(t, filepath.Join(fixture.core, LinuxKitDirectory), got)
}

func TestResolve_FindLinuxKitDirectory_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindLinuxKitDirectory(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindLinuxKitDirectory_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindLinuxKitDirectory(marked, fixture.child)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindLinuxKitManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindLinuxKitManifest(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindLinuxKitManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindLinuxKitManifest(marked, fixture.child)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveLinuxKitManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(filepath.Join(fixture.core, LinuxKitDirectory, FileLinuxKit), "kernel: [broken"))
	got, err := ResolveLinuxKitManifest(fixture.medium, fixture.child)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_WorkspaceSandboxRoot_Good(t *core.T) {
	got := WorkspaceSandboxRoot("repo", "dev")
	core.AssertContains(t, got, filepath.Join(Directory, WorkspaceDirectory, "repo", "dev"))
	core.AssertNotContains(t, got, WorkspaceSourceDirectory)
}

func TestResolve_WorkspaceSandboxRoot_Bad(t *core.T) {
	got := WorkspaceSandboxRoot("../repo", "dev")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestResolve_WorkspaceSandboxRoot_Ugly(t *core.T) {
	got := WorkspaceSandboxRoot("", "")
	core.AssertEqual(t, filepath.Join(core.Env("DIR_HOME"), Directory, WorkspaceDirectory), got)
	core.AssertContains(t, got, WorkspaceDirectory)
}

func TestResolve_WorkspaceSandboxSourcePath_Good(t *core.T) {
	got := WorkspaceSandboxSourcePath("repo", "dev", "app", "main.go")
	core.AssertContains(t, got, filepath.Join(WorkspaceSourceDirectory, "app", "main.go"))
	core.AssertContains(t, got, "repo")
}

func TestResolve_WorkspaceSandboxSourcePath_Bad(t *core.T) {
	got := WorkspaceSandboxSourcePath("repo", "../dev")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestResolve_WorkspaceSandboxSourcePath_Ugly(t *core.T) {
	got := WorkspaceSandboxSourcePath("", "", "")
	core.AssertEqual(t, filepath.Join(core.Env("DIR_HOME"), Directory, WorkspaceDirectory, WorkspaceSourceDirectory), got)
	core.AssertContains(t, got, WorkspaceSourceDirectory)
}

func TestResolve_WorkspaceSandboxMetaPath_Good(t *core.T) {
	got := WorkspaceSandboxMetaPath("repo", "dev", "status.json")
	core.AssertContains(t, got, filepath.Join(WorkspaceMetaDirectory, "status.json"))
	core.AssertContains(t, got, "dev")
}

func TestResolve_WorkspaceSandboxMetaPath_Bad(t *core.T) {
	got := WorkspaceSandboxMetaPath("repo", "dev", "../status.json")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestResolve_WorkspaceSandboxMetaPath_Ugly(t *core.T) {
	got := WorkspaceSandboxMetaPath("", "", "")
	core.AssertEqual(t, filepath.Join(core.Env("DIR_HOME"), Directory, WorkspaceDirectory, WorkspaceMetaDirectory), got)
	core.AssertContains(t, got, WorkspaceMetaDirectory)
}

func TestResolve_WorkspaceSandboxInstructionsPath_Good(t *core.T) {
	got := WorkspaceSandboxInstructionsPath("repo", "dev")
	core.AssertContains(t, got, WorkspaceInstructionsFile)
	core.AssertContains(t, got, "repo")
}

func TestResolve_WorkspaceSandboxInstructionsPath_Bad(t *core.T) {
	got := WorkspaceSandboxInstructionsPath("../repo", "dev")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestResolve_WorkspaceSandboxInstructionsPath_Ugly(t *core.T) {
	got := WorkspaceSandboxInstructionsPath("", "")
	core.AssertEqual(t, filepath.Join(core.Env("DIR_HOME"), Directory, WorkspaceDirectory, WorkspaceInstructionsFile), got)
	core.AssertContains(t, got, WorkspaceInstructionsFile)
}

func TestResolve_WorkspaceSandboxPath_Bad(t *core.T) {
	got := WorkspaceSandboxPath("repo", "dev", "../secret")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestResolve_FindReposManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindReposManifest(m, filepath.Join(t.TempDir(), "repo"))
	core.AssertEqual(t, "", got)
}

func TestResolve_FindReposManifest_Ugly(t *core.T) {
	home := core.Env("DIR_HOME")
	coreDir := filepath.Join(home, "Code", Directory)
	m := symlinkMockMedium{MockMedium: coreio.NewMockMedium(), symlinks: map[string]bool{coreDir: true}}
	core.RequireNoError(t, m.EnsureDir(coreDir))
	core.RequireNoError(t, m.Write(filepath.Join(coreDir, FileRepos), "version: 1\norg: ax\nrepos: []\n"))
	got := FindReposManifest(m, filepath.Join(t.TempDir(), "repo"))
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveReposManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	workspace := filepath.Join(t.TempDir(), "workspace")
	start := filepath.Join(workspace, "repo", "service")
	core.RequireNoError(t, m.EnsureDir(filepath.Join(workspace, Directory)))
	core.RequireNoError(t, m.EnsureDir(start))
	core.RequireNoError(t, m.Write(filepath.Join(workspace, Directory, FileRepos), "version: 1\norg: ax\nrepos:\n  - path: core/go\n    remote: ssh://example/core/go.git\n"))
	got, err := ResolveReposManifest(m, start)
	core.AssertNoError(t, err)
	core.AssertEqual(t, "ax", got.Org)
}

func TestResolve_ResolveReposManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := ResolveReposManifest(m, t.TempDir())
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveReposManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	workspace := filepath.Join(t.TempDir(), "workspace")
	start := filepath.Join(workspace, "repo", "service")
	core.RequireNoError(t, m.EnsureDir(filepath.Join(workspace, Directory)))
	core.RequireNoError(t, m.EnsureDir(start))
	core.RequireNoError(t, m.Write(filepath.Join(workspace, Directory, FileRepos), "version: [broken"))
	got, err := ResolveReposManifest(m, start)
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindPHPManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindPHPManifest(m, t.TempDir())
	core.AssertEqual(t, "", got)
}

func TestResolve_FindPHPManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	marked := symlinkMockMedium{MockMedium: fixture.medium, symlinks: map[string]bool{fixture.core: true}}
	got := FindPHPManifest(marked, fixture.child)
	core.AssertEqual(t, "", got)
}
