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
func ResolveConfigManifest(medium coreio.Medium, start string) core.Result {
	path := FindManifest(medium, start, FileConfig)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveConfigManifest", "no config manifest could be detected", nil))
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
func ResolveBuildManifest(medium coreio.Medium, start string) core.Result {
	path := FindBuildManifest(medium, start)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveBuildManifest", "no build manifest could be detected", nil))
	}

	var build BuildManifest
	if r := LoadManifest(medium, path, &build); !r.OK {
		return r
	}
	return core.Ok(&build)
}

// FindTestManifest returns the nearest project-local .core/test.yaml.
func FindTestManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileTest)
}

// ResolveTestManifest loads the nearest project-local .core/test.yaml or falls
// back to auto-detecting a test command from the repository contents.
func ResolveTestManifest(medium coreio.Medium, start string) core.Result {
	if path := FindTestManifest(medium, start); path != "" {
		var test TestManifest
		if r := LoadManifest(medium, path, &test); !r.OK {
			return r
		}
		return core.Ok(&test)
	}

	detectedResult := detectTestCommand(medium, start)
	if !detectedResult.OK {
		return detectedResult
	}
	detected := detectedResult.Value.(detectedTestCommand)
	if detected.Found {
		return core.Ok(&TestManifest{
			Version: 1,
			Commands: []TestCommand{
				{Name: "test", Run: detected.Command},
			},
		})
	}

	return core.Fail(coreerr.E(callerResolveTestManifest, "no test command could be detected", nil))
}

// FindReleaseManifest returns the nearest project-local .core/release.yaml.
func FindReleaseManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileRelease)
}

// ResolveReleaseManifest loads the nearest project-local .core/release.yaml.
func ResolveReleaseManifest(medium coreio.Medium, start string) core.Result {
	path := FindReleaseManifest(medium, start)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveReleaseManifest", "no release manifest could be detected", nil))
	}

	var release ReleaseManifest
	if r := LoadManifest(medium, path, &release); !r.OK {
		return r
	}
	return core.Ok(&release)
}

// FindRunManifest returns the nearest project-local .core/run.yaml.
func FindRunManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileRun)
}

// ResolveRunManifest loads the nearest project-local .core/run.yaml.
func ResolveRunManifest(medium coreio.Medium, start string) core.Result {
	path := FindRunManifest(medium, start)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveRunManifest", "no run manifest could be detected", nil))
	}

	var run RunManifest
	if r := LoadManifest(medium, path, &run); !r.OK {
		return r
	}
	return core.Ok(&run)
}

// FindViewManifest returns the nearest project-local .core/view.yaml.
func FindViewManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileView)
}

// ResolveViewManifest loads the nearest project-local .core/view.yaml.
func ResolveViewManifest(medium coreio.Medium, start string) core.Result {
	path := FindViewManifest(medium, start)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveViewManifest", "no view manifest could be detected", nil))
	}

	var view ViewManifest
	if r := LoadManifest(medium, path, &view); !r.OK {
		return r
	}
	return core.Ok(&view)
}

// FindPackageManifest returns the nearest project-local .core/manifest.yaml.
func FindPackageManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileManifest)
}

// ResolvePackageManifest loads the nearest project-local .core/manifest.yaml.
func ResolvePackageManifest(medium coreio.Medium, start string) core.Result {
	path := FindPackageManifest(medium, start)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolvePackageManifest", "no package manifest could be detected", nil))
	}

	var pkg PackageManifest
	if r := LoadManifest(medium, path, &pkg); !r.OK {
		return r
	}
	return core.Ok(&pkg)
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
func ResolveAgentManifest(medium coreio.Medium) core.Result {
	path := FindAgentManifest(medium)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveAgentManifest", "no agent manifest could be detected", nil))
	}

	var agent AgentManifest
	if r := LoadManifest(medium, path, &agent); !r.OK {
		return r
	}
	return core.Ok(&agent)
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
func ResolveZoneManifest(medium coreio.Medium) core.Result {
	path := FindZoneManifest(medium)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveZoneManifest", "no zone manifest could be detected", nil))
	}

	var zone ZoneManifest
	if r := LoadManifest(medium, path, &zone); !r.OK {
		return r
	}
	return core.Ok(&zone)
}

// FindWorkspaceManifest returns the nearest project-local .core/workspace.yaml.
func FindWorkspaceManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileWorkspace)
}

// ResolveWorkspaceManifest loads the nearest project-local .core/workspace.yaml.
func ResolveWorkspaceManifest(medium coreio.Medium, start string) core.Result {
	path := FindWorkspaceManifest(medium, start)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveWorkspaceManifest", "no workspace manifest could be detected", nil))
	}

	var ws WorkspaceManifest
	if r := LoadManifest(medium, path, &ws); !r.OK {
		return r
	}
	return core.Ok(&ws)
}

// FindIDEManifest returns the nearest project-local .core/ide.yaml.
func FindIDEManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileIDE)
}

// ResolveIDEManifest loads the nearest project-local .core/ide.yaml.
func ResolveIDEManifest(medium coreio.Medium, start string) core.Result {
	path := FindIDEManifest(medium, start)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveIDEManifest", "no ide manifest could be detected", nil))
	}

	var ide IDEManifest
	if r := LoadManifest(medium, path, &ide); !r.OK {
		return r
	}
	return core.Ok(&ide)
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
func ResolveLinuxKitManifest(medium coreio.Medium, start string) core.Result {
	path := FindLinuxKitManifest(medium, start)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveLinuxKitManifest", "no linuxkit manifest could be detected", nil))
	}

	manifestResult := Load(medium, path)
	if !manifestResult.OK {
		return manifestResult
	}
	return manifestResult
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
func ResolveReposManifest(medium coreio.Medium, start string) core.Result {
	path := FindReposManifest(medium, start)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolveReposManifest", "no repos manifest could be detected", nil))
	}

	var repos ReposManifest
	if r := LoadManifest(medium, path, &repos); !r.OK {
		return r
	}
	return core.Ok(&repos)
}

// ResolveWorkspaceRegistryManifest loads the workspace-root .core/repos.yaml.
// It mirrors ResolveReposManifest for callers that prefer the RFC-aligned
// workspace-registry naming.
func ResolveWorkspaceRegistryManifest(medium coreio.Medium, start string) core.Result {
	return ResolveReposManifest(medium, start)
}

// FindPHPManifest returns the nearest project-local .core/php.yaml.
func FindPHPManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FilePHP)
}

// ResolvePHPManifest loads the nearest project-local .core/php.yaml.
func ResolvePHPManifest(medium coreio.Medium, start string) core.Result {
	path := FindPHPManifest(medium, start)
	if path == "" {
		return core.Fail(coreerr.E("config.ResolvePHPManifest", "no php manifest could be detected", nil))
	}

	var php PHPManifest
	if r := LoadManifest(medium, path, &php); !r.OK {
		return r
	}
	return core.Ok(&php)
}
