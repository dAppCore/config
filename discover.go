package config

import (
	"path/filepath"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

var knownConfigFiles = []string{
	"manifest.yaml",
	"build.yaml",
	"test.yaml",
	"repos.yaml",
	"view.yaml",
	"ide.yaml",
	"config.yaml",
}

// Discover walks from the current directory upward collecting `.core/` directories
// and merges all known config files using closest-wins precedence.
func Discover(opts ...Option) (*Config, error) {
	base, err := newConfig(false, opts...)
	if err != nil {
		return nil, err
	}

	dirs, err := discoverCoreDirs(base.medium)
	if err != nil {
		return nil, err
	}

	// Default to the closest known config path for persistence.
	if len(dirs) > 0 {
		base.path = filepath.Join(dirs[0], "config.yaml")
	}

	for _, coreDir := range dirs {
		cfg, err := discoverConfigForDir(base.medium, coreDir, base.env)
		if err != nil {
			return nil, err
		}
		base.MergeFrom(cfg)
	}

	return base, nil
}

func discoverConfigForDir(m coreio.Medium, coreDir string, env string) (*Config, error) {
	cfg, err := newConfig(false, WithMedium(m), WithPath(filepath.Join(coreDir, "config.yaml")), WithEnvPrefix(env))
	if err != nil {
		return nil, err
	}

	for _, file := range knownConfigFiles {
		filePath := filepath.Join(coreDir, file)
		if !m.IsFile(filePath) {
			continue
		}
		if err := cfg.LoadFile(m, filePath); err != nil {
			return nil, err
		}
	}

	return cfg, nil
}

func discoverCoreDirs(m coreio.Medium) ([]string, error) {
	cwd, err := filepath.Abs(".")
	if err != nil {
		return nil, err
	}

	dirs := make([]string, 0)
	current := filepath.Clean(cwd)
	for {
		corePath := filepath.Join(current, ".core")
		if m.IsDir(corePath) {
			dirs = append(dirs, corePath)
		}

		if m.IsDir(filepath.Join(current, ".git")) {
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	if home := core.Env("DIR_HOME"); home != "" {
		global := filepath.Join(home, ".core")
		if len(dirs) == 0 || filepath.Clean(dirs[len(dirs)-1]) != filepath.Clean(global) {
			dirs = append(dirs, global)
		}
	}

	return dirs, nil
}
