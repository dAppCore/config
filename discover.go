package config

import (
	"os"

	core "dappco.re/go/core"
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
func Discover(opts ...Option) (*Config, error) {
	// os.Getwd is deliberate: core.Env("DIR_CWD") is captured once at init,
	// but callers (including tests) chdir at runtime and need the live CWD.
	cwd, err := os.Getwd()
	if err != nil {
		return nil, coreerr.E("config.Discover", "failed to read working directory", err)
	}
	return DiscoverFrom(cwd, opts...)
}

// DiscoverFrom walks upward from start and builds a merged Config.
// Primarily used by tests; callers usually want Discover().
//
//	cfg, _ := config.DiscoverFrom("/srv/app", config.WithMedium(io.Local))
func DiscoverFrom(start string, opts ...Option) (*Config, error) {
	base, err := newConfig(false, opts...)
	if err != nil {
		return nil, coreerr.E(callerDiscoverFrom, "failed to initialise base config", err)
	}
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
		layer, err := New(layerOpts...)
		if err != nil {
			return nil, coreerr.E(callerDiscoverFrom, "failed to load discovered config: "+p, err)
		}
		base.MergeFrom(layer)
	}
	if err := base.loadStoreState(); err != nil {
		return nil, coreerr.E(callerDiscoverFrom, "failed to load config store state", err)
	}

	return base, nil
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
