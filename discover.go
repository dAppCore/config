package config

import (
	"os"
	"path/filepath"

	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// Discover walks from the current working directory upward, collecting every
// `.core/` directory found, and returns a merged Config with closest-wins
// precedence. Walks stop at the filesystem root or when a `.git/` directory
// is found at the same level as a `.core/` (repository boundary).
//
//	cfg, err := config.Discover()
//	cfg.Get("build.target", &target)  // merged from all .core/ dirs
func Discover(opts ...Option) (*Config, error) {
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
	base, err := New(opts...)
	if err != nil {
		return nil, coreerr.E("config.DiscoverFrom", "failed to initialise base config", err)
	}
	medium := base.medium
	if medium == nil {
		medium = coreio.Local
	}

	paths := discoverPaths(medium, start)

	// paths are ordered closest → furthest; closest-wins via MergeFrom.
	for _, p := range paths {
		layer, err := New(WithMedium(medium), WithPath(p))
		if err != nil {
			return nil, coreerr.E("config.DiscoverFrom", "failed to load discovered config: "+p, err)
		}
		base.MergeFrom(layer)
	}

	return base, nil
}

// discoverPaths returns paths to `.core/config.yaml` files from the starting
// directory up to the filesystem root (or repository boundary), followed by
// the global `~/.core/config.yaml` as the lowest-precedence layer.
func discoverPaths(medium coreio.Medium, start string) []string {
	var paths []string
	dir := start
	for {
		coreDir := filepath.Join(dir, ".core")
		candidate := filepath.Join(coreDir, "config.yaml")
		if medium.Exists(candidate) {
			paths = append(paths, candidate)
		}

		// Repository boundary: stop once a .git sits next to the .core dir.
		if medium.Exists(filepath.Join(dir, ".git")) {
			break
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	if home, err := os.UserHomeDir(); err == nil {
		global := filepath.Join(home, ".core", "config.yaml")
		if medium.Exists(global) && !contains(paths, global) {
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
