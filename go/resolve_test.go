package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"gopkg.in/yaml.v3"
)

const (
	resolveTestWorkspaceReposYAML    = "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"
	resolveTestFailedToParseManifest = "failed to parse manifest"
	resolveTestBrokenVersionYAML     = "version: [broken"
	resolveTestEmptyReposYAML        = "version: 1\norg: host-uk\nrepos: []\n"
	resolveTestMainGo                = "main.go"
	resolveTestStatusJSON            = "status.json"
	resolveTestParentRepoPath        = "../repo"
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
	base := core.PathJoin(tmp, "workspace")
	repo := core.PathJoin(base, "repo")
	child := core.PathJoin(repo, "service")

	for _, dir := range []string{
		core.PathJoin(base, ".core"),
		core.PathJoin(repo, ".core"),
		core.PathJoin(repo, ".git"),
		child,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	globalConfig := core.PathJoin(core.Env("DIR_HOME"), ".core", FileConfig)
	projectConfig := core.PathJoin(repo, ".core", FileConfig)
	workspaceRepos := core.PathJoin(base, ".core", FileRepos)
	core.AssertNoError(t, m.Write(globalConfig, "app:\n  name: global\n"))
	core.AssertNoError(t, m.Write(projectConfig, "app:\n  name: project\n"))
	core.AssertNoError(t, m.Write(workspaceRepos, resolveTestWorkspaceReposYAML))

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
	start := core.PathJoin(t.TempDir(), "missing", "service")

	core.AssertEmpty(t, FindConfigManifest(m, start))
}

func TestResolve_FindProjectManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	base := core.PathJoin(tmp, "workspace")
	repo := core.PathJoin(base, "repo")
	child := core.PathJoin(repo, "service")

	for _, dir := range []string{
		core.PathJoin(base, ".core"),
		core.PathJoin(repo, ".core"),
		core.PathJoin(repo, ".git"),
		child,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}
	core.AssertNoError(t, m.EnsureDir(core.PathJoin(base, ".core")))

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
		core.AssertNoError(t, m.Write(core.PathJoin(repo, ".core", name), content))
	}
	core.AssertNoError(t, m.Write(core.PathJoin(base, ".core", FileBuild), "name: external\noutput: ext\n"))
	core.AssertNoError(t, m.Write(core.PathJoin(base, ".core", FileRepos), resolveTestWorkspaceReposYAML))
	core.AssertEqual(t, core.PathJoin(repo, ".core", FileBuild), FindProjectManifest(m, child, FileBuild))

	cases := []struct {
		name string
		path string
		got  string
	}{
		{name: "build", path: core.PathJoin(repo, ".core", FileBuild), got: FindBuildManifest(m, child)},
		{name: "release", path: core.PathJoin(repo, ".core", FileRelease), got: FindReleaseManifest(m, child)},
		{name: "run", path: core.PathJoin(repo, ".core", FileRun), got: FindRunManifest(m, child)},
		{name: "view", path: core.PathJoin(repo, ".core", FileView), got: FindViewManifest(m, child)},
		{name: "manifest", path: core.PathJoin(repo, ".core", FileManifest), got: FindPackageManifest(m, child)},
		{name: "workspace", path: core.PathJoin(repo, ".core", FileWorkspace), got: FindWorkspaceManifest(m, child)},
		{name: "ide", path: core.PathJoin(repo, ".core", FileIDE), got: FindIDEManifest(m, child)},
		{name: "repos", path: core.PathJoin(base, ".core", FileRepos), got: FindReposManifest(m, child)},
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
	repo := core.PathJoin(tmp, "repo")
	child := core.PathJoin(repo, "service")

	for _, dir := range []string{
		core.PathJoin(repo, ".core"),
		core.PathJoin(repo, ".core", LinuxKitDirectory),
		core.PathJoin(repo, ".git"),
		child,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}
	lkPath := core.PathJoin(repo, ".core", LinuxKitDirectory, FileLinuxKit)
	core.AssertNoError(t, m.Write(lkPath, "version: 1\n"))

	core.AssertEqual(t, lkPath, FindLinuxKitManifest(m, child))
}

func TestResolve_ResolveLinuxKitManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := core.PathJoin(tmp, "repo")
	child := core.PathJoin(repo, "service")

	for _, dir := range []string{
		core.PathJoin(repo, ".core"),
		core.PathJoin(repo, ".core", LinuxKitDirectory),
		core.PathJoin(repo, ".git"),
		child,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}
	core.AssertNoError(t, m.Write(core.PathJoin(repo, ".core", LinuxKitDirectory, FileLinuxKit), "kernel:\n  image: linuxkit/kernel:6.6.0\n"))

	manifest, err := linuxKitManifestResult(ResolveLinuxKitManifest(m, child))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "linuxkit/kernel:6.6.0", manifest["kernel"].(map[string]any)["image"])
}

func TestResolve_ResolveLinuxKitManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	manifest, err := linuxKitManifestResult(ResolveLinuxKitManifest(m, t.TempDir()))
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no linuxkit manifest could be detected")
}

func TestResolve_ResolveTestManifest_Good(t *core.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		expected string
	}{
		{
			name:     "core manifest",
			filename: FileTest,
			content:  "version: 1\ncommands:\n  - name: unit\n    run: vendor/bin/pest --parallel\n",
			expected: "vendor/bin/pest --parallel",
		},
		{
			name:     "composer fallback",
			filename: "composer.json",
			content:  `{"scripts":{}}`,
			expected: "composer test",
		},
		{
			name:     "package script",
			filename: "package.json",
			content:  `{"scripts":{"test":"npm run test:unit"}}`,
			expected: "npm run test:unit",
		},
		{
			name:     "go module",
			filename: "go.mod",
			content:  "module example.com/repo\n",
			expected: "core go qa",
		},
		{
			name:     "pytest ini",
			filename: "pytest.ini",
			content:  "[pytest]\n",
			expected: "pytest",
		},
		{
			name:     "pyproject",
			filename: "pyproject.toml",
			content:  "[tool.pytest.ini_options]\n",
			expected: "pytest",
		},
		{
			name:     "taskfile",
			filename: "Taskfile.yaml",
			content:  "version: '3'\n",
			expected: "task test",
		},
		{
			name:     "taskfile-yml",
			filename: "Taskfile.yml",
			content:  "version: '3'\n",
			expected: "task test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *core.T) {
			m := coreio.NewMockMedium()
			root := core.PathJoin("/", "repo", tc.name)
			child := core.PathJoin(root, "service")

			core.AssertNoError(t, m.EnsureDir(core.PathJoin(root, ".core")))
			core.AssertNoError(t, m.EnsureDir(child))
			core.AssertNoError(t, m.EnsureDir(core.PathJoin(root, ".git")))

			path := core.PathJoin(root, tc.filename)
			if tc.filename == FileTest {
				path = core.PathJoin(root, ".core", FileTest)
			}
			core.AssertNoError(t, m.Write(path, tc.content))

			manifest, err := testManifestResult(ResolveTestManifest(m, child))
			core.AssertNoError(t, err)
			core.AssertNotNil(t, manifest)
			core.AssertNotEmpty(t, manifest.Commands)
			core.AssertEqual(t, tc.expected, manifest.Commands[0].Run)
		})
	}
}

func TestResolve_ResolveTestManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	root := core.PathJoin("/", "repo", "bad")
	child := core.PathJoin(root, "service")

	core.AssertNoError(t, m.EnsureDir(core.PathJoin(root, ".git")))
	core.AssertNoError(t, m.EnsureDir(child))
	core.AssertNoError(t, m.Write(core.PathJoin(root, "package.json"), `{"scripts":{"test":123}}`))

	manifest, err := testManifestResult(ResolveTestManifest(m, child))
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid npm test script")
}

func TestResolve_ResolveTestManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	root := core.PathJoin("/", "repo", "ugly")

	core.AssertNoError(t, m.EnsureDir(core.PathJoin(root, ".git")))

	manifest, err := testManifestResult(ResolveTestManifest(m, root))
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no test command could be detected")
}

func TestResolve_FindProjectManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	base := core.PathJoin(tmp, "workspace")
	repo := core.PathJoin(base, "repo")
	child := core.PathJoin(repo, "service")

	for _, dir := range []string{
		core.PathJoin(base, ".core"),
		core.PathJoin(repo, ".core"),
		core.PathJoin(repo, ".git"),
		child,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	core.AssertEmpty(t, FindProjectManifest(m, child, FileBuild))
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
	start := core.PathJoin(t.TempDir(), "missing", "service")
	core.AssertEmpty(t, FindProjectManifest(nil, start, FileBuild))
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
		core.PathJoin(home, ".core"),
		core.PathJoin(home, ".core", DirectoryImages),
		core.PathJoin(home, ".core", DirectorySecrets),
		core.PathJoin(home, ".core", DirectoryDaemons),
		core.PathJoin(home, ".core", DirectoryWorkspaces),
	} {
		core.RequireNoError(t, m.EnsureDir(dir))
	}

	core.RequireNoError(t, m.Write(core.PathJoin(home, ".core", FileAgent), "daemon:\n  enabled: true\n"))
	core.RequireNoError(t, m.Write(core.PathJoin(home, ".core", FileZone), "zone:\n  name: alpha\n  identity: '@alpha@lthn'\n"))
	core.RequireNoError(t, m.Write(core.PathJoin(home, ".core", DirectoryImages, FileImagesManifest), "{\"images\":{}}"))

	core.AssertEqual(t, core.PathJoin(home, ".core", FileAgent), FindUserManifest(m, FileAgent))
	core.AssertEqual(t, core.PathJoin(home, ".core", FileAgent), FindAgentManifest(m))
	core.AssertEqual(t, core.PathJoin(home, ".core", FileAgent), FindUserPath(m, "", FileAgent))
	core.AssertEqual(t, core.PathJoin(home, ".core", FileZone), FindZoneManifest(m))
	core.AssertEqual(t, core.PathJoin(home, ".core", DirectoryImages), FindUserImagesDirectory(m))
	core.AssertEqual(t, core.PathJoin(home, ".core", DirectoryImages, FileImagesManifest), FindUserImagesManifest(m))
	core.AssertEqual(t, core.PathJoin(home, ".core", DirectoryImages, FileImagesManifest), FindUserPath(m, DirectoryImages, "", FileImagesManifest))
	core.AssertEqual(t, core.PathJoin(home, ".core", DirectorySecrets), FindUserSecretsDirectory(m))
	core.AssertEqual(t, core.PathJoin(home, ".core", DirectoryDaemons), FindUserDaemonsDirectory(m))
	core.AssertEqual(t, core.PathJoin(home, ".core", DirectoryWorkspaces), FindUserWorkspacesDirectory(m))
	core.AssertEqual(t, core.PathJoin(home, ".core", DirectoryImages), FindUserDirectory(m, DirectoryImages))
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
	coreDir := core.PathJoin(home, ".core")
	m := symlinkMockMedium{
		MockMedium: coreio.NewMockMedium(),
		symlinks:   map[string]bool{coreDir: true},
	}
	core.RequireNoError(t, m.EnsureDir(coreDir))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileAgent), "daemon:\n  enabled: true\n"))

	core.AssertEmpty(t, FindUserPath(m, FileAgent))
	core.AssertEmpty(t, FindUserManifest(m, FileAgent))
}

func TestResolve_ResolveAgentManifest_UserManifests_Good(t *core.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(core.PathJoin(home, ".core")))
	core.RequireNoError(t, m.Write(core.PathJoin(home, ".core", FileAgent), "daemon:\n  enabled: true\nagents:\n  worker:\n    total: 2\n"))
	core.RequireNoError(t, m.Write(core.PathJoin(home, ".core", FileZone), "zone:\n  name: alpha\n  identity: '@alpha@lthn'\n  services:\n    vpn:\n      enabled: true\n"))

	agent, err := agentManifestResult(ResolveAgentManifest(m))
	core.RequireNoError(t, err)
	core.RequireTrue(t, agent != nil)
	core.AssertTrue(t, agent.Daemon.Enabled)
	core.AssertEqual(t, 2, agent.Agents["worker"].Total)

	zone, err := zoneManifestResult(ResolveZoneManifest(m))
	core.RequireNoError(t, err)
	core.RequireTrue(t, zone != nil)
	core.AssertEqual(t, "alpha", zone.Zone.Name)
	core.AssertTrue(t, zone.Zone.Services.VPN.Enabled)
}

func TestResolve_ResolveAgentManifest_UserManifests_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	agent, err := agentManifestResult(ResolveAgentManifest(m))
	core.AssertNil(t, agent)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no agent manifest could be detected")

	zone, err := zoneManifestResult(ResolveZoneManifest(m))
	core.AssertNil(t, zone)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no zone manifest could be detected")
}

func TestResolve_ResolveAgentManifest_UserManifests_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(core.PathJoin(home, ".core")))
	core.RequireNoError(t, m.Write(core.PathJoin(home, ".core", FileAgent), "daemon:\n  enabled: [broken"))
	core.RequireNoError(t, m.Write(core.PathJoin(home, ".core", FileZone), "zone:\n  name: [broken"))

	agent, err := agentManifestResult(ResolveAgentManifest(m))
	core.AssertNil(t, agent)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), resolveTestFailedToParseManifest)

	zone, err := zoneManifestResult(ResolveZoneManifest(m))
	core.AssertNil(t, zone)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), resolveTestFailedToParseManifest)
}

func TestResolve_FindPHPManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := core.PathJoin(tmp, "repo")
	child := core.PathJoin(repo, "service")

	core.RequireNoError(t, m.EnsureDir(core.PathJoin(repo, ".core")))
	core.RequireNoError(t, m.EnsureDir(core.PathJoin(repo, ".git")))
	core.RequireNoError(t, m.EnsureDir(child))
	core.RequireNoError(t, m.Write(core.PathJoin(repo, ".core", FilePHP), "version: 1\n"))
	core.RequireNoError(t, m.EnsureDir(core.PathJoin(repo, ".core", LinuxKitDirectory)))

	core.AssertEqual(t, core.PathJoin(repo, ".core", FilePHP), FindPHPManifest(m, child))
	core.AssertEqual(t, core.PathJoin(repo, ".core", LinuxKitDirectory), FindLinuxKitDirectory(m, child))
}

func TestResolve_ResolvePHPManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	php, err := phpManifestResult(ResolvePHPManifest(m, t.TempDir()))
	core.AssertNil(t, php)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no php manifest could be detected")
}

func TestResolve_ResolvePHPManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := core.PathJoin(tmp, "repo")

	core.RequireNoError(t, m.EnsureDir(core.PathJoin(repo, ".core")))
	core.RequireNoError(t, m.Write(core.PathJoin(repo, ".core", FilePHP), resolveTestBrokenVersionYAML))

	php, err := phpManifestResult(ResolvePHPManifest(m, repo))
	core.AssertNil(t, php)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), resolveTestFailedToParseManifest)
}

func TestResolve_ResolvePHPManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := core.PathJoin(tmp, "repo")

	core.RequireNoError(t, m.EnsureDir(core.PathJoin(repo, ".core")))
	core.RequireNoError(t, m.Write(core.PathJoin(repo, ".core", FilePHP), "version: 1\nserver:\n  type: php-fpm\n"))

	php, err := phpManifestResult(ResolvePHPManifest(m, repo))
	core.RequireNoError(t, err)
	core.RequireTrue(t, php != nil)
	core.AssertEqual(t, 1, php.Version)
	core.AssertEqual(t, "php-fpm", php.Server.Type)
}

func TestResolve_FindReposManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	start := core.PathJoin(tmp, "workspace", "repo", "service")
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(start))
	core.RequireNoError(t, m.EnsureDir(core.PathJoin(home, "Code", Directory)))
	core.RequireNoError(t, m.Write(core.PathJoin(home, "Code", Directory, FileRepos), resolveTestEmptyReposYAML))

	core.AssertEqual(t, core.PathJoin(home, "Code", Directory, FileRepos), FindReposManifest(m, start))
}

func TestResolve_FindWorkspaceRegistryManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	start := core.PathJoin(tmp, "workspace", "repo", "service")
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(core.PathDir(core.PathJoin(home, "Code", Directory, FileRepos))))
	core.RequireNoError(t, m.Write(core.PathJoin(home, "Code", Directory, FileRepos), resolveTestEmptyReposYAML))

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
	coreDir := core.PathJoin(home, "Code", Directory)
	start := core.PathJoin("workspace", "repo", "service")
	m := symlinkMockMedium{
		MockMedium: coreio.NewMockMedium(),
		symlinks:   map[string]bool{coreDir: true},
	}
	core.RequireNoError(t, m.EnsureDir(coreDir))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileRepos), resolveTestEmptyReposYAML))

	core.AssertEmpty(t, FindWorkspaceRegistryManifest(m, start))
}

func TestResolve_ResolveWorkspaceRegistryManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	start := core.PathJoin(tmp, "workspace", "repo", "service")
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(core.PathJoin(home, "Code", Directory)))
	core.RequireNoError(t, m.Write(core.PathJoin(home, "Code", Directory, FileRepos), resolveTestWorkspaceReposYAML))

	repos, err := reposManifestResult(ResolveWorkspaceRegistryManifest(m, start))
	core.RequireNoError(t, err)
	core.RequireTrue(t, repos != nil)
	core.AssertEqual(t, "host-uk", repos.Org)
	core.AssertLen(t, repos.Repos, 1)
}

func TestResolve_ResolveWorkspaceRegistryManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	repos, err := reposManifestResult(ResolveWorkspaceRegistryManifest(m, t.TempDir()))
	core.AssertNil(t, repos)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no repos manifest could be detected")
}

func TestResolve_ResolveWorkspaceRegistryManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := core.PathJoin(tmp, "workspace", "repo", "service")

	core.RequireNoError(t, m.EnsureDir(core.PathJoin(home, "Code", Directory)))
	core.RequireNoError(t, m.Write(core.PathJoin(home, "Code", Directory, FileRepos), resolveTestBrokenVersionYAML))

	repos, err := reposManifestResult(ResolveWorkspaceRegistryManifest(m, start))
	core.AssertNil(t, repos)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), resolveTestFailedToParseManifest)
}

func TestResolve_ResolveImagesManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")

	core.RequireNoError(t, m.EnsureDir(core.PathJoin(home, ".core", DirectoryImages)))
	core.RequireNoError(t, m.Write(core.PathJoin(home, ".core", DirectoryImages, FileImagesManifest), `{"images":{"core-dev":{"version":"1.2.3","downloaded":"2026-04-15T12:00:00Z","source":"github"}}}`))

	manifest, err := imagesManifestResult(ResolveImagesManifest(m))
	core.RequireNoError(t, err)
	core.RequireTrue(t, manifest != nil)
	core.AssertLen(t, manifest.Images, 1)
	core.AssertEqual(t, "1.2.3", manifest.Images["core-dev"].Version)
}

func TestResolve_WorkspaceSandboxPath_Good(t *core.T) {
	home := core.Env("DIR_HOME")
	core.AssertEqual(t, core.PathJoin(home, Directory, WorkspaceDirectory, "repo", "dev", "src"), WorkspaceSandboxPath("repo", "dev", "src"))
	core.AssertEqual(t, core.PathJoin(home, Directory, WorkspaceDirectory, "repo", "dev"), WorkspaceSandboxRoot("repo", "dev"))
	core.AssertEqual(t, core.PathJoin(home, Directory, WorkspaceDirectory, "repo", "dev", WorkspaceSourceDirectory, "app", resolveTestMainGo), WorkspaceSandboxSourcePath("repo", "dev", "app", resolveTestMainGo))
	core.AssertEqual(t, core.PathJoin(home, Directory, WorkspaceDirectory, "repo", "dev", WorkspaceMetaDirectory, resolveTestStatusJSON), WorkspaceSandboxMetaPath("repo", "dev", resolveTestStatusJSON))
	core.AssertEqual(t, core.PathJoin(home, Directory, WorkspaceDirectory, "repo", "dev", WorkspaceInstructionsFile), WorkspaceSandboxInstructionsPath("repo", "dev"))
}

func TestResolve_WorkspaceSandboxPath_Ugly(t *core.T) {
	home := core.Env("DIR_HOME")
	core.AssertEqual(t, core.PathJoin(home, Directory, WorkspaceDirectory, "src"), WorkspaceSandboxPath("", "", "", "src", ""))
	core.AssertEmpty(t, WorkspaceSandboxPath(resolveTestParentRepoPath, "dev"))
	core.AssertEmpty(t, WorkspaceSandboxPath("repo", "../dev"))
	core.AssertEmpty(t, WorkspaceSandboxPath("repo", "dev", "../secret"))
}

func TestResolve_ResolveConfigManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := core.PathJoin(tmp, "workspace", "repo", "service")

	for _, dir := range []string{
		core.PathJoin(home, ".core"),
		start,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	configPath := core.PathJoin(home, ".core", FileConfig)
	core.AssertNoError(t, m.Write(configPath, "app:\n  name: global\n"))

	cfg, err := configResult(ResolveConfigManifest(m, start))
	core.AssertNoError(t, err)
	core.AssertNotNil(t, cfg)
	core.AssertEqual(t, configPath, cfg.Path())

	var name string
	core.AssertNoError(t, resultError(cfg.Get("app.name", &name)))
	core.AssertEqual(t, "global", name)
}

func TestResolve_ResolveConfigManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	start := core.PathJoin(t.TempDir(), "workspace", "repo", "service")

	_, err := configResult(ResolveConfigManifest(m, start))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no config manifest could be detected")
}

func TestResolve_ResolveConfigManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := core.PathJoin(tmp, "workspace", "repo", "service")

	for _, dir := range []string{
		core.PathJoin(home, ".core"),
		start,
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	configPath := core.PathJoin(home, ".core", FileConfig)
	core.AssertNoError(t, m.Write(configPath, "app:\n  name: [broken yaml"))

	_, err := configResult(ResolveConfigManifest(m, start))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to load config file")
}

func TestResolve_ResolveBuildManifest_ProjectManifests_Good(t *core.T) {
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := core.PathJoin(tmp, "workspace", "repo")
	workspace := core.PathJoin(tmp, "workspace")
	child := core.PathJoin(repo, "service")

	for _, dir := range []string{
		core.PathJoin(repo, ".core"),
		core.PathJoin(repo, ".git"),
		child,
		core.PathJoin(workspace, ".core"),
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	buildPath := core.PathJoin(repo, ".core", FileBuild)
	releasePath := core.PathJoin(repo, ".core", FileRelease)
	runPath := core.PathJoin(repo, ".core", FileRun)
	viewPath := core.PathJoin(repo, ".core", FileView)
	packagePath := core.PathJoin(repo, ".core", FileManifest)
	idePath := core.PathJoin(repo, ".core", FileIDE)

	core.AssertNoError(t, m.Write(buildPath, "name: core\noutput: dist\ntargets:\n  - linux/amd64\n"))
	core.AssertNoError(t, m.Write(releasePath, "version: 1\narchive:\n  format: tar.gz\n  include:\n    - README.md\nchecksums: true\ngithub:\n  draft: false\n  prerelease: false\nchangelog:\n  include:\n    - feat\n"))
	core.AssertNoError(t, m.Write(runPath, "version: 1\nservices:\n  - name: db\n    image: postgres:16\n    port: 5432\ndev:\n  command: php artisan serve\n  port: 8000\n  watch:\n    - app/\n"))
	core.AssertNoError(t, m.Write(viewPath, "code: photo-browser\nname: Photo Browser\nsign: "+base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))+"\npermissions:\n  clipboard: true\n"))
	core.AssertNoError(t, m.Write(packagePath, packageManifestFixture(t)))
	core.AssertNoError(t, m.Write(idePath, "version: 1\neditor: nvim\n"))
	core.AssertNoError(t, m.Write(core.PathJoin(workspace, ".core", FileRepos), resolveTestWorkspaceReposYAML))

	build, err := buildManifestResult(ResolveBuildManifest(m, child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", build.Name)
	core.AssertEqual(t, "dist", build.Output)

	release, err := releaseManifestResult(ResolveReleaseManifest(m, child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, true, release.Checksums)
	core.AssertEqual(t, "tar.gz", release.Archive.Format)

	run, err := runManifestResult(ResolveRunManifest(m, child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "php artisan serve", run.Dev.Command)
	core.AssertLen(t, run.Services, 1)

	view, err := viewManifestResult(ResolveViewManifest(m, child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "photo-browser", view.Code)
	core.AssertTrue(t, view.Permissions.Clipboard)

	pkg, err := packageManifestResult(ResolvePackageManifest(m, child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "go-io", pkg.Code)
	core.AssertEqual(t, "Core I/O", pkg.Name)

	ide, err := ideManifestResult(ResolveIDEManifest(m, child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, ide.Version)
	core.AssertEqual(t, "nvim", ide.Editor)

	repos, err := reposManifestResult(ResolveReposManifest(m, child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "host-uk", repos.Org)
	core.AssertLen(t, repos.Repos, 1)
}

func TestResolve_ResolveBuildManifest_ProjectManifests_Bad(t *core.T) {
	t.Setenv("DIR_HOME", t.TempDir())
	m := coreio.NewMockMedium()
	start := core.PathJoin(t.TempDir(), "workspace", "repo", "service")

	_, err := buildManifestResult(ResolveBuildManifest(m, start))
	core.AssertError(t, err)
	_, err = releaseManifestResult(ResolveReleaseManifest(m, start))
	core.AssertError(t, err)
	_, err = runManifestResult(ResolveRunManifest(m, start))
	core.AssertError(t, err)
	_, err = viewManifestResult(ResolveViewManifest(m, start))
	core.AssertError(t, err)
	_, err = packageManifestResult(ResolvePackageManifest(m, start))
	core.AssertError(t, err)
	_, err = ideManifestResult(ResolveIDEManifest(m, start))
	core.AssertError(t, err)
	_, err = reposManifestResult(ResolveReposManifest(m, start))
	core.AssertError(t, err)
}

func TestResolve_FindReposManifest_FallsBackToWorkspaceRoot_Good(t *core.T) {
	m := coreio.NewMockMedium()
	start := core.PathJoin(t.TempDir(), "workspace", "repo", "service")
	reposPath := core.PathJoin(core.Env("DIR_HOME"), "Code", ".core", FileRepos)

	core.AssertNoError(t, m.EnsureDir(core.PathDir(reposPath)))
	core.AssertNoError(t, m.Write(reposPath, resolveTestWorkspaceReposYAML))

	core.AssertEqual(t, reposPath, FindReposManifest(m, start))
}

func TestResolve_ResolveBuildManifest_ProjectManifests_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := core.PathJoin(tmp, "workspace", "repo")
	workspace := core.PathJoin(tmp, "workspace")
	child := core.PathJoin(repo, "service")

	for _, dir := range []string{
		core.PathJoin(repo, ".core"),
		core.PathJoin(repo, ".git"),
		child,
		core.PathJoin(workspace, ".core"),
	} {
		core.AssertNoError(t, m.EnsureDir(dir))
	}

	core.AssertNoError(t, m.Write(core.PathJoin(repo, ".core", FileBuild), "name: core\noutput: dist\ntargets:\n  - [broken yaml"))
	core.AssertNoError(t, m.Write(core.PathJoin(repo, ".core", FileRelease), "version: 1\narchive:\n  format: [broken yaml"))
	core.AssertNoError(t, m.Write(core.PathJoin(repo, ".core", FileRun), "version: 1\nservices: [broken yaml"))
	core.AssertNoError(t, m.Write(core.PathJoin(repo, ".core", FileView), "code: photo-browser\nname: Photo Browser\npermissions:\n  clipboard: true\n"))
	core.AssertNoError(t, m.Write(core.PathJoin(repo, ".core", FileManifest), "code: go-io\nname: Core I/O\nsign: not-base64\nsign_key: \"\"\n"))
	core.AssertNoError(t, m.Write(core.PathJoin(workspace, ".core", FileRepos), "version: 1\norg: host-uk\nrepos: [broken yaml"))

	_, err := buildManifestResult(ResolveBuildManifest(m, child))
	core.AssertError(t, err)
	_, err = releaseManifestResult(ResolveReleaseManifest(m, child))
	core.AssertError(t, err)
	_, err = runManifestResult(ResolveRunManifest(m, child))
	core.AssertError(t, err)

	_, err = viewManifestResult(ResolveViewManifest(m, child))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsigned view manifest rejected")

	_, err = packageManifestResult(ResolvePackageManifest(m, child))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "missing package sign_key")

	_, err = reposManifestResult(ResolveReposManifest(m, child))
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
	msg, err := bytesResult(packageManifestBytes(pkg))
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
	root := core.PathJoin(t.TempDir(), "repo")
	child := core.PathJoin(root, "service")
	coreDir := core.PathJoin(root, Directory)
	core.RequireNoError(t, m.EnsureDir(coreDir))
	core.RequireNoError(t, m.EnsureDir(core.PathJoin(root, ".git")))
	core.RequireNoError(t, m.EnsureDir(child))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileBuild), "version: 1\nproject:\n  name: core\nbuild:\n  flags: [-trimpath]\ntargets:\n  - linux/amd64\n"))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileTest), "version: 1\ncommands:\n  - name: unit\n    run: go test ./...\n"))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileRelease), "version: 1\narchive:\n  format: tar.gz\n  include: [README.md]\nchecksums: true\n"))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileRun), "version: 1\nservices:\n  - name: db\n    image: postgres:16\n    port: 5432\n"))
	view, _ := axSignedView(t)
	viewBody, err := yaml.Marshal(&view)
	core.RequireNoError(t, err)
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileView), string(viewBody)))
	pkg, pub := axSignedPackage(t)
	setManifestTrustKeys(t, hex.EncodeToString(pub))
	pkgBody, err := yaml.Marshal(&pkg)
	core.RequireNoError(t, err)
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileManifest), string(pkgBody)))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileWorkspace), "version: 1\ndependencies: [core/go]\n"))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileIDE), "version: 1\neditor: codex\n"))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FilePHP), "version: 1\nserver:\n  type: php-fpm\n"))
	core.RequireNoError(t, m.EnsureDir(core.PathJoin(coreDir, LinuxKitDirectory)))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, LinuxKitDirectory, FileLinuxKit), "kernel:\n  image: linuxkit/kernel:6.6.0\n"))
	return axResolveProject{medium: m, root: root, child: child, core: coreDir}
}

func axResolveUserFixture(t *core.T) (*coreio.MockMedium, string) {
	t.Helper()
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")
	coreDir := core.PathJoin(home, Directory)
	core.RequireNoError(t, m.EnsureDir(coreDir))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileAgent), "daemon:\n  enabled: true\nagents:\n  codex:\n    total: 1\n"))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileZone), "zone:\n  name: alpha\n  identity: '@alpha@lthn'\n  services:\n    vpn:\n      enabled: true\n"))
	for _, dir := range []string{DirectoryImages, DirectorySecrets, DirectoryDaemons, DirectoryWorkspaces} {
		core.RequireNoError(t, m.EnsureDir(core.PathJoin(coreDir, dir)))
	}
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, DirectoryImages, FileImagesManifest), `{"images":{}}`))
	return m, home
}

func TestResolve_FindUserManifest_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserManifest(m, FileAgent)
	core.AssertEqual(t, core.PathJoin(home, Directory, FileAgent), got)
}

func TestResolve_FindUserManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserManifest(m, FileAgent)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{core.PathJoin(home, Directory): true}}
	got := FindUserManifest(marked, FileAgent)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserDirectory_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserDirectory(m, DirectoryImages)
	core.AssertEqual(t, core.PathJoin(home, Directory, DirectoryImages), got)
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
	core.AssertEqual(t, core.PathJoin(home, Directory, DirectoryImages, FileImagesManifest), got)
}

func TestResolve_FindUserImagesManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserImagesManifest(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserImagesManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{core.PathJoin(home, Directory): true}}
	got := FindUserImagesManifest(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserImagesDirectory_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserImagesDirectory(m)
	core.AssertEqual(t, core.PathJoin(home, Directory, DirectoryImages), got)
}

func TestResolve_FindUserImagesDirectory_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserImagesDirectory(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserImagesDirectory_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{core.PathJoin(home, Directory): true}}
	got := FindUserImagesDirectory(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserSecretsDirectory_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserSecretsDirectory(m)
	core.AssertEqual(t, core.PathJoin(home, Directory, DirectorySecrets), got)
}

func TestResolve_FindUserSecretsDirectory_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserSecretsDirectory(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserSecretsDirectory_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{core.PathJoin(home, Directory): true}}
	got := FindUserSecretsDirectory(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserDaemonsDirectory_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserDaemonsDirectory(m)
	core.AssertEqual(t, core.PathJoin(home, Directory, DirectoryDaemons), got)
}

func TestResolve_FindUserDaemonsDirectory_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserDaemonsDirectory(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserDaemonsDirectory_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{core.PathJoin(home, Directory): true}}
	got := FindUserDaemonsDirectory(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserWorkspacesDirectory_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindUserWorkspacesDirectory(m)
	core.AssertEqual(t, core.PathJoin(home, Directory, DirectoryWorkspaces), got)
}

func TestResolve_FindUserWorkspacesDirectory_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindUserWorkspacesDirectory(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindUserWorkspacesDirectory_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{core.PathJoin(home, Directory): true}}
	got := FindUserWorkspacesDirectory(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindBuildManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindBuildManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, core.PathJoin(fixture.core, FileBuild), got)
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
	got, err := buildManifestResult(ResolveBuildManifest(fixture.medium, fixture.child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", got.Project.Name)
}

func TestResolve_ResolveBuildManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := buildManifestResult(ResolveBuildManifest(m, t.TempDir()))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveBuildManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(core.PathJoin(fixture.core, FileBuild), "targets:\n  - invalid-target\n"))
	got, err := buildManifestResult(ResolveBuildManifest(fixture.medium, fixture.child))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindTestManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindTestManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, core.PathJoin(fixture.core, FileTest), got)
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
	core.AssertEqual(t, core.PathJoin(fixture.core, FileRelease), got)
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
	got, err := releaseManifestResult(ResolveReleaseManifest(fixture.medium, fixture.child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "tar.gz", got.Archive.Format)
}

func TestResolve_ResolveReleaseManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := releaseManifestResult(ResolveReleaseManifest(m, t.TempDir()))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveReleaseManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(core.PathJoin(fixture.core, FileRelease), resolveTestBrokenVersionYAML))
	got, err := releaseManifestResult(ResolveReleaseManifest(fixture.medium, fixture.child))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindRunManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindRunManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, core.PathJoin(fixture.core, FileRun), got)
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
	got, err := runManifestResult(ResolveRunManifest(fixture.medium, fixture.child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "db", got.Services[0].Name)
}

func TestResolve_ResolveRunManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := runManifestResult(ResolveRunManifest(m, t.TempDir()))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveRunManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(core.PathJoin(fixture.core, FileRun), "services: [broken"))
	got, err := runManifestResult(ResolveRunManifest(fixture.medium, fixture.child))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindViewManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindViewManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, core.PathJoin(fixture.core, FileView), got)
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
	got, err := viewManifestResult(ResolveViewManifest(fixture.medium, fixture.child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "photo-browser", got.Code)
}

func TestResolve_ResolveViewManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := viewManifestResult(ResolveViewManifest(m, t.TempDir()))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveViewManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(core.PathJoin(fixture.core, FileView), "code: ax\nname: AX\n"))
	got, err := viewManifestResult(ResolveViewManifest(fixture.medium, fixture.child))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindPackageManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindPackageManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, core.PathJoin(fixture.core, FileManifest), got)
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
	got, err := packageManifestResult(ResolvePackageManifest(fixture.medium, fixture.child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "go-config", got.Code)
}

func TestResolve_ResolvePackageManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := packageManifestResult(ResolvePackageManifest(m, t.TempDir()))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolvePackageManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(core.PathJoin(fixture.core, FileManifest), "code: ax\nname: AX\nsign: not-base64\nsign_key: \"\"\n"))
	got, err := packageManifestResult(ResolvePackageManifest(fixture.medium, fixture.child))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindAgentManifest_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindAgentManifest(m)
	core.AssertEqual(t, core.PathJoin(home, Directory, FileAgent), got)
}

func TestResolve_FindAgentManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindAgentManifest(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindAgentManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{core.PathJoin(home, Directory): true}}
	got := FindAgentManifest(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveAgentManifest_Good(t *core.T) {
	m, _ := axResolveUserFixture(t)
	got, err := agentManifestResult(ResolveAgentManifest(m))
	core.AssertNoError(t, err)
	core.AssertTrue(t, got.Daemon.Enabled)
}

func TestResolve_ResolveAgentManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := agentManifestResult(ResolveAgentManifest(m))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveAgentManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	core.RequireNoError(t, m.Write(core.PathJoin(home, Directory, FileAgent), "daemon: [broken"))
	got, err := agentManifestResult(ResolveAgentManifest(m))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindZoneManifest_Good(t *core.T) {
	m, home := axResolveUserFixture(t)
	got := FindZoneManifest(m)
	core.AssertEqual(t, core.PathJoin(home, Directory, FileZone), got)
}

func TestResolve_FindZoneManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindZoneManifest(m)
	core.AssertEqual(t, "", got)
}

func TestResolve_FindZoneManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	marked := symlinkMockMedium{MockMedium: m, symlinks: map[string]bool{core.PathJoin(home, Directory): true}}
	got := FindZoneManifest(marked)
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveZoneManifest_Good(t *core.T) {
	m, _ := axResolveUserFixture(t)
	got, err := zoneManifestResult(ResolveZoneManifest(m))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "alpha", got.Zone.Name)
}

func TestResolve_ResolveZoneManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := zoneManifestResult(ResolveZoneManifest(m))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveZoneManifest_Ugly(t *core.T) {
	m, home := axResolveUserFixture(t)
	core.RequireNoError(t, m.Write(core.PathJoin(home, Directory, FileZone), "zone: [broken"))
	got, err := zoneManifestResult(ResolveZoneManifest(m))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindWorkspaceManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	root := core.PathJoin("/", "workspace", "repo")
	child := core.PathJoin(root, "service")

	core.AssertNoError(t, m.EnsureDir(core.PathJoin(root, ".core")))
	core.AssertNoError(t, m.EnsureDir(child))
	core.AssertNoError(t, m.Write(core.PathJoin(root, ".core", FileWorkspace), "version: 1\ndependencies:\n  - core-php\nactive: core-php\npackages_dir: ./packages\n"))

	path := FindWorkspaceManifest(m, child)
	core.AssertEqual(t, core.PathJoin(root, ".core", FileWorkspace), path)
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

func TestResolve_ResolveWorkspaceManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	root := core.PathJoin("/", "workspace", "resolve")

	core.AssertNoError(t, m.EnsureDir(core.PathJoin(root, ".core")))
	core.AssertNoError(t, m.Write(core.PathJoin(root, ".core", FileWorkspace), "version: 1\ndependencies:\n  - core-php\nactive: core-php\npackages_dir: ./packages\nsettings:\n  suggest_core_commands: true\n"))

	manifest, err := workspaceManifestResult(ResolveWorkspaceManifest(m, root))
	core.AssertNoError(t, err)
	core.AssertNotNil(t, manifest)
	core.AssertEqual(t, []string{"core-php"}, manifest.Dependencies)
	core.AssertEqual(t, "core-php", manifest.Active)
	core.AssertEqual(t, "./packages", manifest.PackagesDir)
	core.AssertEqual(t, true, manifest.Settings["suggest_core_commands"])
}

func TestResolve_ResolveWorkspaceManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()

	manifest, err := workspaceManifestResult(ResolveWorkspaceManifest(m, core.PathJoin("/", "workspace", "missing")))
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no workspace manifest could be detected")
}

func TestResolve_ResolveWorkspaceManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	root := core.PathJoin("/", "workspace", "ugly")

	core.AssertNoError(t, m.EnsureDir(core.PathJoin(root, ".core")))
	core.AssertNoError(t, m.Write(core.PathJoin(root, ".core", FileWorkspace), "version: [broken yaml"))

	manifest, err := workspaceManifestResult(ResolveWorkspaceManifest(m, root))
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
}

func TestResolve_FindIDEManifest_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindIDEManifest(fixture.medium, fixture.child)
	core.AssertEqual(t, core.PathJoin(fixture.core, FileIDE), got)
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
	got, err := ideManifestResult(ResolveIDEManifest(fixture.medium, fixture.child))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "codex", got.Editor)
}

func TestResolve_ResolveIDEManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := ideManifestResult(ResolveIDEManifest(m, t.TempDir()))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveIDEManifest_Ugly(t *core.T) {
	fixture := axResolveProjectFixture(t)
	core.RequireNoError(t, fixture.medium.Write(core.PathJoin(fixture.core, FileIDE), resolveTestBrokenVersionYAML))
	got, err := ideManifestResult(ResolveIDEManifest(fixture.medium, fixture.child))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_FindLinuxKitDirectory_Good(t *core.T) {
	fixture := axResolveProjectFixture(t)
	got := FindLinuxKitDirectory(fixture.medium, fixture.child)
	core.AssertEqual(t, core.PathJoin(fixture.core, LinuxKitDirectory), got)
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
	core.RequireNoError(t, fixture.medium.Write(core.PathJoin(fixture.core, LinuxKitDirectory, FileLinuxKit), "kernel: [broken"))
	got, err := linuxKitManifestResult(ResolveLinuxKitManifest(fixture.medium, fixture.child))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_WorkspaceSandboxRoot_Good(t *core.T) {
	got := WorkspaceSandboxRoot("repo", "dev")
	core.AssertContains(t, got, core.PathJoin(Directory, WorkspaceDirectory, "repo", "dev"))
	core.AssertNotContains(t, got, WorkspaceSourceDirectory)
}

func TestResolve_WorkspaceSandboxRoot_Bad(t *core.T) {
	got := WorkspaceSandboxRoot(resolveTestParentRepoPath, "dev")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestResolve_WorkspaceSandboxRoot_Ugly(t *core.T) {
	got := WorkspaceSandboxRoot("", "")
	core.AssertEqual(t, core.PathJoin(core.Env("DIR_HOME"), Directory, WorkspaceDirectory), got)
	core.AssertContains(t, got, WorkspaceDirectory)
}

func TestResolve_WorkspaceSandboxSourcePath_Good(t *core.T) {
	got := WorkspaceSandboxSourcePath("repo", "dev", "app", resolveTestMainGo)
	core.AssertContains(t, got, core.PathJoin(WorkspaceSourceDirectory, "app", resolveTestMainGo))
	core.AssertContains(t, got, "repo")
}

func TestResolve_WorkspaceSandboxSourcePath_Bad(t *core.T) {
	got := WorkspaceSandboxSourcePath("repo", "../dev")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestResolve_WorkspaceSandboxSourcePath_Ugly(t *core.T) {
	got := WorkspaceSandboxSourcePath("", "", "")
	core.AssertEqual(t, core.PathJoin(core.Env("DIR_HOME"), Directory, WorkspaceDirectory, WorkspaceSourceDirectory), got)
	core.AssertContains(t, got, WorkspaceSourceDirectory)
}

func TestResolve_WorkspaceSandboxMetaPath_Good(t *core.T) {
	got := WorkspaceSandboxMetaPath("repo", "dev", resolveTestStatusJSON)
	core.AssertContains(t, got, core.PathJoin(WorkspaceMetaDirectory, resolveTestStatusJSON))
	core.AssertContains(t, got, "dev")
}

func TestResolve_WorkspaceSandboxMetaPath_Bad(t *core.T) {
	got := WorkspaceSandboxMetaPath("repo", "dev", "../status.json")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestResolve_WorkspaceSandboxMetaPath_Ugly(t *core.T) {
	got := WorkspaceSandboxMetaPath("", "", "")
	core.AssertEqual(t, core.PathJoin(core.Env("DIR_HOME"), Directory, WorkspaceDirectory, WorkspaceMetaDirectory), got)
	core.AssertContains(t, got, WorkspaceMetaDirectory)
}

func TestResolve_WorkspaceSandboxInstructionsPath_Good(t *core.T) {
	got := WorkspaceSandboxInstructionsPath("repo", "dev")
	core.AssertContains(t, got, WorkspaceInstructionsFile)
	core.AssertContains(t, got, "repo")
}

func TestResolve_WorkspaceSandboxInstructionsPath_Bad(t *core.T) {
	got := WorkspaceSandboxInstructionsPath(resolveTestParentRepoPath, "dev")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestResolve_WorkspaceSandboxInstructionsPath_Ugly(t *core.T) {
	got := WorkspaceSandboxInstructionsPath("", "")
	core.AssertEqual(t, core.PathJoin(core.Env("DIR_HOME"), Directory, WorkspaceDirectory, WorkspaceInstructionsFile), got)
	core.AssertContains(t, got, WorkspaceInstructionsFile)
}

func TestResolve_WorkspaceSandboxPath_Bad(t *core.T) {
	got := WorkspaceSandboxPath("repo", "dev", "../secret")
	core.AssertEqual(t, "", got)
	core.AssertEmpty(t, got)
}

func TestResolve_FindReposManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got := FindReposManifest(m, core.PathJoin(t.TempDir(), "repo"))
	core.AssertEqual(t, "", got)
}

func TestResolve_FindReposManifest_Ugly(t *core.T) {
	home := core.Env("DIR_HOME")
	coreDir := core.PathJoin(home, "Code", Directory)
	m := symlinkMockMedium{MockMedium: coreio.NewMockMedium(), symlinks: map[string]bool{coreDir: true}}
	core.RequireNoError(t, m.EnsureDir(coreDir))
	core.RequireNoError(t, m.Write(core.PathJoin(coreDir, FileRepos), "version: 1\norg: ax\nrepos: []\n"))
	got := FindReposManifest(m, core.PathJoin(t.TempDir(), "repo"))
	core.AssertEqual(t, "", got)
}

func TestResolve_ResolveReposManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	workspace := core.PathJoin(t.TempDir(), "workspace")
	start := core.PathJoin(workspace, "repo", "service")
	core.RequireNoError(t, m.EnsureDir(core.PathJoin(workspace, Directory)))
	core.RequireNoError(t, m.EnsureDir(start))
	core.RequireNoError(t, m.Write(core.PathJoin(workspace, Directory, FileRepos), "version: 1\norg: ax\nrepos:\n  - path: core/go\n    remote: ssh://example/core/go.git\n"))
	got, err := reposManifestResult(ResolveReposManifest(m, start))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "ax", got.Org)
}

func TestResolve_ResolveReposManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	got, err := reposManifestResult(ResolveReposManifest(m, t.TempDir()))
	core.AssertNil(t, got)
	core.AssertError(t, err)
}

func TestResolve_ResolveReposManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	workspace := core.PathJoin(t.TempDir(), "workspace")
	start := core.PathJoin(workspace, "repo", "service")
	core.RequireNoError(t, m.EnsureDir(core.PathJoin(workspace, Directory)))
	core.RequireNoError(t, m.EnsureDir(start))
	core.RequireNoError(t, m.Write(core.PathJoin(workspace, Directory, FileRepos), resolveTestBrokenVersionYAML))
	got, err := reposManifestResult(ResolveReposManifest(m, start))
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
