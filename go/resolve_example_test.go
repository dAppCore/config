package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"gopkg.in/yaml.v3"
)

func exampleResolveMedium() (*coreio.MockMedium, string, func()) {
	m := coreio.NewMockMedium()
	workspace := core.PathJoin("/", "workspace")
	root := core.PathJoin(workspace, "repo")
	child := core.PathJoin(root, "service")
	home := core.Env("DIR_HOME")
	cleanup := func() {}

	for _, dir := range []string{
		core.PathJoin(workspace, Directory),
		core.PathJoin(root, Directory),
		core.PathJoin(root, Directory, LinuxKitDirectory),
		core.PathJoin(root, ".git"),
		child,
		core.PathJoin(home, Directory),
		core.PathJoin(home, Directory, DirectoryImages),
		core.PathJoin(home, Directory, DirectorySecrets),
		core.PathJoin(home, Directory, DirectoryDaemons),
		core.PathJoin(home, Directory, DirectoryWorkspaces),
	} {
		_ = m.EnsureDir(dir)
	}

	view, _ := exampleSignedViewManifest()
	viewBody, _ := yaml.Marshal(&view)
	pkg, trustCleanup := exampleSignedPackageManifest()
	cleanup = trustCleanup
	pkgBody, _ := yaml.Marshal(&pkg)

	_ = m.Write(core.PathJoin(root, Directory, FileConfig), "app:\n  name: project\n")
	_ = m.Write(core.PathJoin(root, Directory, FileBuild), "version: 1\nproject:\n  name: app\n  main: ./cmd/app\nbuild:\n  flags: [-trimpath]\ntargets:\n  - linux/amd64\n")
	_ = m.Write(core.PathJoin(root, Directory, FileTest), "version: 1\ncommands:\n  - name: unit\n    run: go test ./...\n")
	_ = m.Write(core.PathJoin(root, Directory, FileRelease), "version: 1\narchive:\n  format: tar.gz\n  include:\n    - README.md\nchecksums: true\ngithub:\n  draft: false\n  prerelease: false\nchangelog:\n  include:\n    - feat\n")
	_ = m.Write(core.PathJoin(root, Directory, FileRun), "version: 1\nservices:\n  - name: db\n    image: postgres:16\n    port: 5432\ndev:\n  command: go run ./cmd/app\n  port: 8080\n")
	_ = m.Write(core.PathJoin(root, Directory, FileView), string(viewBody))
	_ = m.Write(core.PathJoin(root, Directory, FileManifest), string(pkgBody))
	_ = m.Write(core.PathJoin(root, Directory, FileWorkspace), "version: 1\ndependencies:\n  - core-go\nactive: core-go\npackages_dir: ./packages\n")
	_ = m.Write(core.PathJoin(root, Directory, FileIDE), "version: 1\neditor: nvim\n")
	_ = m.Write(core.PathJoin(root, Directory, FilePHP), "version: 1\nserver:\n  type: php-fpm\n")
	_ = m.Write(core.PathJoin(root, Directory, LinuxKitDirectory, FileLinuxKit), "kernel:\n  image: linuxkit/kernel:6.6.0\n")
	_ = m.Write(core.PathJoin(workspace, Directory, FileRepos), "version: 1\norg: core\nrepos:\n  - path: core/config\n    remote: ssh://example/core/config.git\n")
	_ = m.Write(core.PathJoin(home, Directory, FileAgent), "daemon:\n  enabled: true\nagents:\n  codex:\n    total: 1\n")
	_ = m.Write(core.PathJoin(home, Directory, FileZone), "zone:\n  name: alpha\n  identity: alpha-id\n")
	_ = SaveImagesManifest(m, core.PathJoin(home, Directory, DirectoryImages, FileImagesManifest), &ImagesManifest{Images: map[string]ImageInfo{}})

	return m, child, cleanup
}

func ExampleFindProjectManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindProjectManifest(m, child, FileBuild)))
	// Output: build.yaml
}

func ExampleFindUserManifest() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindUserManifest(m, FileAgent)))
	// Output: agent.yaml
}

func ExampleFindUserPath() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindUserPath(m, FileAgent)))
	// Output: agent.yaml
}

func ExampleFindUserDirectory() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindUserDirectory(m, DirectoryImages)))
	// Output: images
}

func ExampleFindUserImagesManifest() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindUserImagesManifest(m)))
	// Output: manifest.json
}

func ExampleFindUserImagesDirectory() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindUserImagesDirectory(m)))
	// Output: images
}

func ExampleFindUserSecretsDirectory() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindUserSecretsDirectory(m)))
	// Output: secrets
}

func ExampleFindUserDaemonsDirectory() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindUserDaemonsDirectory(m)))
	// Output: daemons
}

func ExampleFindUserWorkspacesDirectory() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindUserWorkspacesDirectory(m)))
	// Output: workspaces
}

func ExampleResolveConfigManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	cfg, err := configResult(ResolveConfigManifest(m, child))
	var name string
	_ = cfg.Get("app.name", &name)
	core.Println(err == nil, name)
	// Output: true project
}

func ExampleFindConfigManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindConfigManifest(m, child)))
	// Output: config.yaml
}

func ExampleFindBuildManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindBuildManifest(m, child)))
	// Output: build.yaml
}

func ExampleResolveBuildManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	build, err := buildManifestResult(ResolveBuildManifest(m, child))
	core.Println(err == nil, build.Project.Name)
	// Output: true app
}

func ExampleFindTestManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindTestManifest(m, child)))
	// Output: test.yaml
}

func ExampleResolveTestManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	test, err := testManifestResult(ResolveTestManifest(m, child))
	core.Println(err == nil, test.Commands[0].Name)
	// Output: true unit
}

func ExampleFindReleaseManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindReleaseManifest(m, child)))
	// Output: release.yaml
}

func ExampleResolveReleaseManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	release, err := releaseManifestResult(ResolveReleaseManifest(m, child))
	core.Println(err == nil, release.Archive.Format)
	// Output: true tar.gz
}

func ExampleFindRunManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindRunManifest(m, child)))
	// Output: run.yaml
}

func ExampleResolveRunManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	run, err := runManifestResult(ResolveRunManifest(m, child))
	core.Println(err == nil, run.Dev.Port)
	// Output: true 8080
}

func ExampleFindViewManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindViewManifest(m, child)))
	// Output: view.yaml
}

func ExampleResolveViewManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	view, err := viewManifestResult(ResolveViewManifest(m, child))
	core.Println(err == nil, view.Code)
	// Output: true app
}

func ExampleFindPackageManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindPackageManifest(m, child)))
	// Output: manifest.yaml
}

func ExampleResolvePackageManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	pkg, err := packageManifestResult(ResolvePackageManifest(m, child))
	core.Println(err == nil, pkg.Code)
	// Output: true go-config
}

func ExampleFindAgentManifest() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindAgentManifest(m)))
	// Output: agent.yaml
}

func ExampleResolveAgentManifest() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	agent, err := agentManifestResult(ResolveAgentManifest(m))
	core.Println(err == nil, agent.Daemon.Enabled)
	// Output: true true
}

func ExampleFindZoneManifest() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindZoneManifest(m)))
	// Output: zone.yaml
}

func ExampleResolveZoneManifest() {
	m, _, cleanup := exampleResolveMedium()
	defer cleanup()
	zone, err := zoneManifestResult(ResolveZoneManifest(m))
	core.Println(err == nil, zone.Zone.Name)
	// Output: true alpha
}

func ExampleFindWorkspaceManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindWorkspaceManifest(m, child)))
	// Output: workspace.yaml
}

func ExampleResolveWorkspaceManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	workspace, err := workspaceManifestResult(ResolveWorkspaceManifest(m, child))
	core.Println(err == nil, workspace.Active)
	// Output: true core-go
}

func ExampleFindIDEManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindIDEManifest(m, child)))
	// Output: ide.yaml
}

func ExampleResolveIDEManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	ide, err := ideManifestResult(ResolveIDEManifest(m, child))
	core.Println(err == nil, ide.Editor)
	// Output: true nvim
}

func ExampleFindLinuxKitDirectory() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindLinuxKitDirectory(m, child)))
	// Output: linuxkit
}

func ExampleFindLinuxKitManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindLinuxKitManifest(m, child)))
	// Output: core-dev.yml
}

func ExampleResolveLinuxKitManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	manifest, err := linuxKitManifestResult(ResolveLinuxKitManifest(m, child))
	core.Println(err == nil, manifest["kernel"].(map[string]any)["image"])
	// Output: true linuxkit/kernel:6.6.0
}

func ExampleWorkspaceSandboxRoot() {
	core.Println(core.PathBase(WorkspaceSandboxRoot("repo", "dev")))
	// Output: dev
}

func ExampleWorkspaceSandboxSourcePath() {
	core.Println(core.PathBase(WorkspaceSandboxSourcePath("repo", "dev", "app", "main.go")))
	// Output: main.go
}

func ExampleWorkspaceSandboxMetaPath() {
	core.Println(core.PathBase(WorkspaceSandboxMetaPath("repo", "dev", "status.json")))
	// Output: status.json
}

func ExampleWorkspaceSandboxInstructionsPath() {
	core.Println(core.PathBase(WorkspaceSandboxInstructionsPath("repo", "dev")))
	// Output: CODEX.md
}

func ExampleWorkspaceSandboxPath() {
	core.Println(core.PathBase(WorkspaceSandboxPath("repo", "dev", ".meta", "status")))
	// Output: status
}

func ExampleFindReposManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindReposManifest(m, child)))
	// Output: repos.yaml
}

func ExampleFindWorkspaceRegistryManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindWorkspaceRegistryManifest(m, child)))
	// Output: repos.yaml
}

func ExampleResolveReposManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	repos, err := reposManifestResult(ResolveReposManifest(m, child))
	core.Println(err == nil, repos.Org)
	// Output: true core
}

func ExampleResolveWorkspaceRegistryManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	repos, err := reposManifestResult(ResolveWorkspaceRegistryManifest(m, child))
	core.Println(err == nil, repos.Org)
	// Output: true core
}

func ExampleFindPHPManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	core.Println(core.PathBase(FindPHPManifest(m, child)))
	// Output: php.yaml
}

func ExampleResolvePHPManifest() {
	m, child, cleanup := exampleResolveMedium()
	defer cleanup()
	php, err := phpManifestResult(ResolvePHPManifest(m, child))
	core.Println(err == nil, php.Server.Type)
	// Output: true php-fpm
}
