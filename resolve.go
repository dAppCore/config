package config

import (
	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
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

func projectCoreDirs(medium coreio.Medium, start string) []string {
	if medium == nil {
		medium = coreio.Local
	}

	var dirs []string
	dir := start
	for {
		coreDir := core.Path(dir, Directory)
		if medium.Exists(coreDir) {
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

	return nil, coreerr.E("config.ResolveTestManifest", "no test command could be detected", nil)
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

// FindReposManifest returns the nearest project-local .core/repos.yaml.
func FindReposManifest(medium coreio.Medium, start string) string {
	return FindProjectManifest(medium, start, FileRepos)
}

// ResolveReposManifest loads the nearest project-local .core/repos.yaml.
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
