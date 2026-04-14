package config

import (
	core "dappco.re/go/core"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
)

type XDGPaths struct {
	prefix string
}

// XDG returns a platform-aware path resolver.
func XDG() *XDGPaths {
	return &XDGPaths{prefix: "core"}
}

func (x *XDGPaths) prefixValue() string {
	if x == nil || x.prefix == "" {
		return "core"
	}
	return x.prefix
}

func (x *XDGPaths) Config() string {
	if configured := core.Env("XDG_CONFIG_HOME"); configured != "" {
		return filepath.Join(configured, x.prefixValue())
	}

	if runtime.GOOS == "windows" {
		root := core.Env("APPDATA")
		if root == "" {
			root = core.Path(core.Env("DIR_HOME"), "AppData", "Roaming")
		}
		return filepath.Join(root, x.prefixValue())
	}

	root := filepath.Join(core.Env("DIR_HOME"), ".config")
	return filepath.Join(root, x.prefixValue())
}

func (x *XDGPaths) Data() string {
	if configured := core.Env("XDG_DATA_HOME"); configured != "" {
		return filepath.Join(configured, x.prefixValue())
	}

	if runtime.GOOS == "windows" {
		root := core.Env("LOCALAPPDATA")
		if root == "" {
			root = core.Path(core.Env("DIR_HOME"), "AppData", "Local")
		}
		return filepath.Join(root, x.prefixValue())
	}

	root := filepath.Join(core.Env("DIR_HOME"), ".local", "share")
	return filepath.Join(root, x.prefixValue())
}

func (x *XDGPaths) Cache() string {
	if configured := core.Env("XDG_CACHE_HOME"); configured != "" {
		return filepath.Join(configured, x.prefixValue())
	}

	if runtime.GOOS == "windows" {
		root := core.Env("LOCALAPPDATA")
		if root == "" {
			root = core.Path(core.Env("DIR_HOME"), "AppData", "Local")
		}
		return filepath.Join(root, "cache", x.prefixValue())
	}

	root := filepath.Join(core.Env("DIR_HOME"), ".cache")
	return filepath.Join(root, x.prefixValue())
}

func (x *XDGPaths) Runtime() string {
	if configured := core.Env("XDG_RUNTIME_DIR"); configured != "" {
		return filepath.Join(configured, x.prefixValue())
	}

	if runtime.GOOS == "windows" {
		root := core.Env("LOCALAPPDATA")
		if root == "" {
			root = core.Path(core.Env("DIR_HOME"), "AppData", "Local")
		}
		return filepath.Join(root, "temp", x.prefixValue())
	}

	if runtime.GOOS == "darwin" {
		root := core.Path(core.Env("DIR_HOME"), "Library", "Caches")
		return filepath.Join(root, x.prefixValue())
	}

	return filepath.Join("/run/user", runtimeUID(), x.prefixValue())
}

func runtimeUID() string {
	if raw := core.Env("UID"); raw != "" {
		if uid, err := strconv.Atoi(raw); err == nil {
			return strconv.Itoa(uid)
		}
	}

	u, err := user.Current()
	if err != nil || u == nil {
		return "1000"
	}
	return u.Uid
}
