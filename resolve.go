package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

// FindProjectManifest searches upward from start for the nearest project-local
// .core manifest file with the given name. The walk stops at the filesystem
// root or a repository boundary and does not consult ~/.core/.
func FindProjectManifest(medium coreio.Medium, start string, name string) string {
	if medium == nil {
		medium = coreio.Local
	}
	for _, dir := range projectCoreDirs(medium, start) {
		candidate := core.Path(dir, name)
		if medium.Exists(candidate) {
			return candidate
		}
	}
	return ""
}

// FindUserManifest returns the user-global ~/.core/{name} path when it exists.
//
//	path := config.FindUserManifest(io.Local, config.FileAgent)
func FindUserManifest(medium coreio.Medium, name string) string {
	return FindUserPath(medium, name)
}

// FindUserPath returns the user-global ~/.core/... path when it exists.
//
//	path := config.FindUserPath(io.Local, config.DirectoryImages, config.FileImagesManifest)
func FindUserPath(medium coreio.Medium, parts ...string) string {
	if medium == nil {
		medium = coreio.Local
	}
	if home := core.Env("DIR_HOME"); home != "" && isSymlinkedCoreDir(medium, core.Path(home, Directory)) {
		return ""
	}
	candidate := userCorePath(parts...)
	if candidate == "" {
		return ""
	}
	if medium.Exists(candidate) {
		return candidate
	}
	return ""
}

// FindUserDirectory returns the user-global ~/.core/<name>/ directory when it exists.
//
//	dir := config.FindUserDirectory(io.Local, config.DirectoryImages)
func FindUserDirectory(medium coreio.Medium, name string) string {
	return FindUserPath(medium, name)
}

// FindUserImagesManifest returns the user-global ~/.core/images/manifest.json path when it exists.
//
//	path := config.FindUserImagesManifest(io.Local)
func FindUserImagesManifest(medium coreio.Medium) string {
	return FindUserPath(medium, DirectoryImages, FileImagesManifest)
}

// FindUserImagesDirectory returns the user-global ~/.core/images/ directory
// when it exists.
//
//	dir := config.FindUserImagesDirectory(io.Local)
func FindUserImagesDirectory(medium coreio.Medium) string {
	return FindUserPath(medium, DirectoryImages)
}

// FindUserSecretsDirectory returns the user-global ~/.core/secrets/ directory
// when it exists.
//
//	dir := config.FindUserSecretsDirectory(io.Local)
func FindUserSecretsDirectory(medium coreio.Medium) string {
	return FindUserPath(medium, DirectorySecrets)
}

// FindUserDaemonsDirectory returns the user-global ~/.core/daemons/ directory
// when it exists.
//
//	dir := config.FindUserDaemonsDirectory(io.Local)
func FindUserDaemonsDirectory(medium coreio.Medium) string {
	return FindUserPath(medium, DirectoryDaemons)
}

// FindUserWorkspacesDirectory returns the user-global ~/.core/workspaces/
// directory when it exists.
//
//	dir := config.FindUserWorkspacesDirectory(io.Local)
func FindUserWorkspacesDirectory(medium coreio.Medium) string {
	return FindUserPath(medium, DirectoryWorkspaces)
}

func projectCoreDirs(medium coreio.Medium, start string) []string {
	if medium == nil {
		medium = coreio.Local
	}

	var dirs []string
	dir := normalizeUpwardStart(medium, start)
	for {
		coreDir := core.Path(dir, Directory)
		if medium.Exists(coreDir) && !isSymlinkedCoreDir(medium, coreDir) {
			dirs = append(dirs, coreDir)
		}
		if medium.Exists(core.Path(dir, ".git")) {
			break
		}
		parent := core.PathDir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return dirs
}

// normalizeUpwardStart converts a file path to its containing directory before
// performing an upward discovery walk. Directory inputs are returned unchanged.
func normalizeUpwardStart(medium coreio.Medium, start string) string {
	if medium == nil {
		medium = coreio.Local
	}
	if start == "" {
		return "."
	}
	if info, err := medium.Stat(start); err == nil && !info.IsDir() {
		return core.PathDir(start)
	}
	return start
}

// userCorePath joins a path under ~/.core/ using DIR_HOME as the home root.
// Empty parts are ignored so callers can build declarative registry paths.
func userCorePath(parts ...string) string {
	home := core.Env("DIR_HOME")
	if home == "" {
		return ""
	}
	elems := make([]string, 0, 2+len(parts))
	elems = append(elems, home, Directory)
	for _, part := range parts {
		if part == "" {
			continue
		}
		if !isSafePathElement(part) {
			return ""
		}
		elems = append(elems, part)
	}
	return core.Path(elems...)
}

// ResolveConfigManifest loads the nearest .core/config.yaml found while
// walking upward from start. Unlike the other manifest helpers, config.yaml is
// also valid at ~/.core/config.yaml, so it uses the broader manifest search.
func ResolveConfigManifest(medium coreio.Medium, start string) (*Config, error) {
	path := FindManifest(medium, start, FileConfig)
	if path == "" {
		return nil, coreerr.E("config.ResolveConfigManifest", "no config manifest could be detected", nil)
	}
	return New(WithMedium(medium), WithPath(path))
}

// FindConfigManifest returns the nearest config.yaml, including the user-global
// ~/.core/config.yaml fallback.
func FindConfigManifest(medium coreio.Medium, start string) string {
	return FindManifest(medium, start, FileConfig)
}

// FindBuildManifest returns the nearest project-local .core/build.yaml.
func FindBuildManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileBuild)
}

// ResolveBuildManifest loads the nearest project-local .core/build.yaml.
func ResolveBuildManifest(medium coreio.Medium, start string) (*BuildManifest, error) {
	path := FindBuildManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolveBuildManifest", "no build manifest could be detected", nil)
	}

	var build BuildManifest
	if err := LoadManifest(medium, path, &build); err != nil {
		return nil, err
	}
	return &build, nil
}

// FindTestManifest returns the nearest project-local .core/test.yaml.
func FindTestManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileTest)
}

// ResolveTestManifest loads the nearest project-local .core/test.yaml or falls
// back to auto-detecting a test command from the repository contents.
func ResolveTestManifest(medium coreio.Medium, start string) (*TestManifest, error) {
	if path := FindTestManifest(medium, start); path != "" {
		var test TestManifest
		if err := LoadManifest(medium, path, &test); err != nil {
			return nil, err
		}
		return &test, nil
	}

	if command, ok, err := detectTestCommand(medium, start); err != nil {
		return nil, err
	} else if ok {
		return &TestManifest{
			Version: 1,
			Commands: []TestCommand{
				{Name: "test", Run: command},
			},
		}, nil
	}

	return nil, coreerr.E(callerResolveTestManifest, "no test command could be detected", nil)
}

// FindReleaseManifest returns the nearest project-local .core/release.yaml.
func FindReleaseManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileRelease)
}

// ResolveReleaseManifest loads the nearest project-local .core/release.yaml.
func ResolveReleaseManifest(medium coreio.Medium, start string) (*ReleaseManifest, error) {
	path := FindReleaseManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolveReleaseManifest", "no release manifest could be detected", nil)
	}

	var release ReleaseManifest
	if err := LoadManifest(medium, path, &release); err != nil {
		return nil, err
	}
	return &release, nil
}

// FindRunManifest returns the nearest project-local .core/run.yaml.
func FindRunManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileRun)
}

// ResolveRunManifest loads the nearest project-local .core/run.yaml.
func ResolveRunManifest(medium coreio.Medium, start string) (*RunManifest, error) {
	path := FindRunManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolveRunManifest", "no run manifest could be detected", nil)
	}

	var run RunManifest
	if err := LoadManifest(medium, path, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

// FindViewManifest returns the nearest project-local .core/view.yaml.
func FindViewManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileView)
}

// ResolveViewManifest loads the nearest project-local .core/view.yaml.
func ResolveViewManifest(medium coreio.Medium, start string) (*ViewManifest, error) {
	path := FindViewManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolveViewManifest", "no view manifest could be detected", nil)
	}

	var view ViewManifest
	if err := LoadManifest(medium, path, &view); err != nil {
		return nil, err
	}
	return &view, nil
}

// FindPackageManifest returns the nearest project-local .core/manifest.yaml.
func FindPackageManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileManifest)
}

// ResolvePackageManifest loads the nearest project-local .core/manifest.yaml.
func ResolvePackageManifest(medium coreio.Medium, start string) (*PackageManifest, error) {
	path := FindPackageManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolvePackageManifest", "no package manifest could be detected", nil)
	}

	var pkg PackageManifest
	if err := LoadManifest(medium, path, &pkg); err != nil {
		return nil, err
	}
	return &pkg, nil
}

// FindAgentManifest returns the user-global ~/.core/agent.yaml when it exists.
//
//	path := config.FindAgentManifest(io.Local)
func FindAgentManifest(medium coreio.Medium) string {
	return FindUserManifest(medium, FileAgent)
}

// ResolveAgentManifest loads the user-global ~/.core/agent.yaml.
//
//	agent, err := config.ResolveAgentManifest(io.Local)
func ResolveAgentManifest(medium coreio.Medium) (*AgentManifest, error) {
	path := FindAgentManifest(medium)
	if path == "" {
		return nil, coreerr.E("config.ResolveAgentManifest", "no agent manifest could be detected", nil)
	}

	var agent AgentManifest
	if err := LoadManifest(medium, path, &agent); err != nil {
		return nil, err
	}
	return &agent, nil
}

// FindZoneManifest returns the user-global ~/.core/zone.yaml when it exists.
//
//	path := config.FindZoneManifest(io.Local)
func FindZoneManifest(medium coreio.Medium) string {
	return FindUserManifest(medium, FileZone)
}

// ResolveZoneManifest loads the user-global ~/.core/zone.yaml.
//
//	zone, err := config.ResolveZoneManifest(io.Local)
func ResolveZoneManifest(medium coreio.Medium) (*ZoneManifest, error) {
	path := FindZoneManifest(medium)
	if path == "" {
		return nil, coreerr.E("config.ResolveZoneManifest", "no zone manifest could be detected", nil)
	}

	var zone ZoneManifest
	if err := LoadManifest(medium, path, &zone); err != nil {
		return nil, err
	}
	return &zone, nil
}

// FindWorkspaceManifest returns the nearest project-local .core/workspace.yaml.
func FindWorkspaceManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileWorkspace)
}

// ResolveWorkspaceManifest loads the nearest project-local .core/workspace.yaml.
func ResolveWorkspaceManifest(medium coreio.Medium, start string) (*WorkspaceManifest, error) {
	path := FindWorkspaceManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolveWorkspaceManifest", "no workspace manifest could be detected", nil)
	}

	var ws WorkspaceManifest
	if err := LoadManifest(medium, path, &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

// FindIDEManifest returns the nearest project-local .core/ide.yaml.
func FindIDEManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileIDE)
}

// ResolveIDEManifest loads the nearest project-local .core/ide.yaml.
func ResolveIDEManifest(medium coreio.Medium, start string) (*IDEManifest, error) {
	path := FindIDEManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolveIDEManifest", "no ide manifest could be detected", nil)
	}

	var ide IDEManifest
	if err := LoadManifest(medium, path, &ide); err != nil {
		return nil, err
	}
	return &ide, nil
}

// FindLinuxKitDirectory returns the nearest project-local .core/linuxkit/
// directory.
//
//	dir := config.FindLinuxKitDirectory(io.Local, cwd)
func FindLinuxKitDirectory(medium coreio.Medium, start string) string {
	return findProjectDirectory(medium, start, LinuxKitDirectory)
}

// FindLinuxKitManifest returns the nearest project-local .core/linuxkit/core-dev.yml
// path when it exists.
//
//	path := config.FindLinuxKitManifest(io.Local, cwd)
func FindLinuxKitManifest(medium coreio.Medium, start string) string {
	if medium == nil {
		medium = coreio.Local
	}
	for _, dir := range projectCoreDirs(medium, start) {
		candidate := core.Path(dir, LinuxKitDirectory, FileLinuxKit)
		if medium.Exists(candidate) {
			return candidate
		}
	}
	return ""
}

// ResolveLinuxKitManifest loads the nearest project-local
// .core/linuxkit/core-dev.yml into a generic map. LinuxKit files are part of
// the .core registry but intentionally stay schema-light in this package.
//
//	lk, err := config.ResolveLinuxKitManifest(io.Local, cwd)
func ResolveLinuxKitManifest(medium coreio.Medium, start string) (map[string]any, error) {
	path := FindLinuxKitManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolveLinuxKitManifest", "no linuxkit manifest could be detected", nil)
	}

	manifest, err := Load(medium, path)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

// findProjectDirectory returns the nearest project-local .core/{name}/
// directory while walking upward from start.
func findProjectDirectory(medium coreio.Medium, start string, name string) string {
	if medium == nil {
		medium = coreio.Local
	}
	if !isSafePathElement(name) {
		return ""
	}
	for _, dir := range projectCoreDirs(medium, start) {
		candidate := core.Path(dir, name)
		if medium.Exists(candidate) {
			return candidate
		}
	}
	return ""
}

// WorkspaceSandboxRoot returns the project-local sandbox root used for agent workspaces.
//
//	root := config.WorkspaceSandboxRoot("my-repo", "dev")
func WorkspaceSandboxRoot(repo, branch string) string {
	return WorkspaceSandboxPath(repo, branch)
}

// WorkspaceSandboxSourcePath returns a path inside the checked-out source tree
// for a sandboxed workspace.
//
//	src := config.WorkspaceSandboxSourcePath("my-repo", "dev", "app", "main.go")
func WorkspaceSandboxSourcePath(repo, branch string, parts ...string) string {
	return WorkspaceSandboxPath(repo, branch, append([]string{WorkspaceSourceDirectory}, parts...)...)
}

// WorkspaceSandboxMetaPath returns a path inside the sandbox metadata
// directory.
//
//	meta := config.WorkspaceSandboxMetaPath("my-repo", "dev", "status.json")
func WorkspaceSandboxMetaPath(repo, branch string, parts ...string) string {
	return WorkspaceSandboxPath(repo, branch, append([]string{WorkspaceMetaDirectory}, parts...)...)
}

// WorkspaceSandboxInstructionsPath returns the agent instruction file for a
// sandboxed workspace.
//
//	path := config.WorkspaceSandboxInstructionsPath("my-repo", "dev")
func WorkspaceSandboxInstructionsPath(repo, branch string) string {
	return WorkspaceSandboxPath(repo, branch, WorkspaceInstructionsFile)
}

// WorkspaceSandboxPath returns a path inside the project-local sandbox workspace tree.
//
//	meta := config.WorkspaceSandboxPath("my-repo", "dev", ".meta", "status")
func WorkspaceSandboxPath(repo, branch string, parts ...string) string {
	elems := []string{Directory, WorkspaceDirectory}
	for _, part := range []string{repo, branch} {
		if part == "" {
			continue
		}
		if !isSafePathElement(part) {
			return ""
		}
		elems = append(elems, part)
	}
	for _, part := range parts {
		if part == "" {
			continue
		}
		if !isSafePathElement(part) {
			return ""
		}
		elems = append(elems, part)
	}
	return core.Path(elems...)
}

// FindReposManifest returns the nearest workspace-root .core/repos.yaml while
// walking upward from start without stopping at repository boundaries. This
// keeps repos.yaml at the shared workspace root rather than under ~/.core/.
func FindReposManifest(medium coreio.Medium, start string) string {
	if medium == nil {
		medium = coreio.Local
	}

	dir := core.CleanPath(normalizeUpwardStart(medium, start), string(core.PathSeparator))
	if dir == "" {
		dir = "."
	}
	for {
		candidate := core.Path(dir, Directory, FileRepos)
		if medium.Exists(candidate) && !isSymlinkedCoreDir(medium, core.Path(dir, Directory)) {
			return candidate
		}
		parent := core.PathDir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if home := core.Env("DIR_HOME"); home != "" {
		// Fallback to the conventional ~/Code/.core/repos.yaml location used by
		// the federated monorepo workspace convention when no repos.yaml is found
		// in the upward walk.
		coreDir := core.Path(home, "Code", Directory)
		candidate := core.Path(coreDir, FileRepos)
		if medium.Exists(candidate) && !isSymlinkedCoreDir(medium, coreDir) {
			return candidate
		}
	}
	return ""
}

// FindWorkspaceRegistryManifest returns the nearest workspace-root
// .core/repos.yaml. It is an alias for FindReposManifest and exists so callers
// can use the workspace-registry naming from the RFC without changing the
// underlying storage layout.
func FindWorkspaceRegistryManifest(medium coreio.Medium, start string) string {
	return FindReposManifest(medium, start)
}

// ResolveReposManifest loads the workspace-root .core/repos.yaml.
func ResolveReposManifest(medium coreio.Medium, start string) (*ReposManifest, error) {
	path := FindReposManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolveReposManifest", "no repos manifest could be detected", nil)
	}

	var repos ReposManifest
	if err := LoadManifest(medium, path, &repos); err != nil {
		return nil, err
	}
	return &repos, nil
}

// ResolveWorkspaceRegistryManifest loads the workspace-root .core/repos.yaml.
// It mirrors ResolveReposManifest for callers that prefer the RFC-aligned
// workspace-registry naming.
func ResolveWorkspaceRegistryManifest(medium coreio.Medium, start string) (*ReposManifest, error) {
	return ResolveReposManifest(medium, start)
}

// FindPHPManifest returns the nearest project-local .core/php.yaml.
func FindPHPManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FilePHP)
}

// ResolvePHPManifest loads the nearest project-local .core/php.yaml.
func ResolvePHPManifest(medium coreio.Medium, start string) (*PHPManifest, error) {
	path := FindPHPManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolvePHPManifest", "no php manifest could be detected", nil)
	}

	var php PHPManifest
	if err := LoadManifest(medium, path, &php); err != nil {
		return nil, err
	}
	return &php, nil
}
