package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

const callerDiscoverFrom = "config.DiscoverFrom"

// Discover walks from the current working directory upward, collecting every
// `.core/` directory found, and returns a merged Config with closest-wins
// precedence. Walks stop at the filesystem root or when a `.git/` directory
// is found at the same level as a `.core/` (repository boundary).
//
//	cfg, err := config.Discover()
//	cfg.Get("build.target", &target)  // merged from all .core/ dirs
func Discover(opts ...Option) core.Result {
	r := core.Getwd()
	if !r.OK {
		return core.Fail(coreerr.E("config.Discover", "failed to read working directory", resultCause(r).(error)))
	}
	return DiscoverFrom(r.Value.(string), opts...)
}

// DiscoverFrom walks upward from start and builds a merged Config.
// Primarily used by tests; callers usually want Discover().
//
//	cfg, _ := config.DiscoverFrom("/srv/app", config.WithMedium(io.Local))
func DiscoverFrom(start string, opts ...Option) core.Result {
	baseResult := newConfig(false, opts...)
	if !baseResult.OK {
		return core.Fail(coreerr.E(callerDiscoverFrom, "failed to initialise base config", resultCause(baseResult).(error)))
	}
	base := baseResult.Value.(*Config)
	medium := base.medium
	if medium == nil {
		medium = coreio.Local
	}
	envPrefix := envPrefixOf(base.full)

	paths := discoverPaths(medium, start)

	// paths are ordered closest → furthest; closest-wins via MergeFrom.
	for _, p := range paths {
		layerOpts := []Option{WithMedium(medium), WithPath(p)}
		if envPrefix != "" {
			layerOpts = append(layerOpts, WithEnvPrefix(envPrefix))
		}
		layerResult := New(layerOpts...)
		if !layerResult.OK {
			return core.Fail(coreerr.E(callerDiscoverFrom, "failed to load discovered config: "+p, resultCause(layerResult).(error)))
		}
		layer := layerResult.Value.(*Config)
		base.MergeFrom(layer)
	}
	if r := base.loadStoreState(); !r.OK {
		return core.Fail(coreerr.E(callerDiscoverFrom, "failed to load config store state", resultCause(r).(error)))
	}

	return core.Ok(base)
}

// discoverPaths returns paths to `.core/config.yaml` files from the starting
// directory up to the filesystem root (or repository boundary), followed by
// the global `~/.core/config.yaml` as the lowest-precedence layer.
func discoverPaths(medium coreio.Medium, start string) []string {
	var paths []string
	dir := normalizeUpwardStart(medium, start)
	for {
		coreDir := core.Path(dir, ".core")
		if !isSymlinkedCoreDir(medium, coreDir) {
			candidate := core.Path(coreDir, "config.yaml")
			if medium.Exists(candidate) {
				paths = append(paths, candidate)
			}
		}

		// Repository boundary: stop once a .git sits next to the .core dir.
		if medium.Exists(core.Path(dir, ".git")) {
			break
		}

		parent := core.PathDir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if home := core.Env("DIR_HOME"); home != "" {
		globalCore := core.Path(home, ".core")
		global := core.Path(globalCore, "config.yaml")
		if !isSymlinkedCoreDir(medium, globalCore) && medium.Exists(global) && !contains(paths, global) {
			paths = append(paths, global)
		}
	}

	return paths
}

func contains(list []string, value string) bool {
	for _, s := range list {
		if s == value {
			return true
		}
	}
	return false
}

// CoreDirs walks upward from start and returns every .core/ directory found,
// closest first. The walk stops at the filesystem root or when a .git sits at
// the same level as a .core. The user-global ~/.core/ is appended last.
//
//	dirs := config.CoreDirs(io.Local, cwd)
//	for _, dir := range dirs { /* check for build.yaml, test.yaml, ... */ }
func CoreDirs(medium coreio.Medium, start string) []string {
	if medium == nil {
		medium = coreio.Local
	}
	var dirs []string
	dir := normalizeUpwardStart(medium, start)
	for {
		coreDir := core.Path(dir, ".core")
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
	if home := core.Env("DIR_HOME"); home != "" {
		global := core.Path(home, ".core")
		if medium.Exists(global) && !isSymlinkedCoreDir(medium, global) && !contains(dirs, global) {
			dirs = append(dirs, global)
		}
	}
	return dirs
}

// FindManifest searches .core/ directories walking up from start for the first
// existing file with the given name (e.g. config.FileBuild). Returns the full
// path or an empty string if none is found.
//
//	path := config.FindManifest(io.Local, cwd, config.FileBuild)
//	if path != "" { config.LoadManifest(io.Local, path, &build) }
func FindManifest(medium coreio.Medium, start string, name string) string {
	if medium == nil {
		medium = coreio.Local
	}
	if !isSafePathElement(name) {
		return ""
	}
	for _, dir := range CoreDirs(medium, start) {
		candidate := core.Path(dir, name)
		if medium.Exists(candidate) {
			return candidate
		}
	}
	return ""
}
