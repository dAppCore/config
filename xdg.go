package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// XDGPaths resolves platform-aware directories following the XDG Base Directory
// Specification on Linux, with sensible defaults for macOS and Windows.
// All paths are suffixed with the application prefix (default "core").
//
//	paths := config.XDG()
//	paths.Config()   // ~/.config/core on Linux
//	paths.Data()     // ~/.local/share/core on Linux
//	paths.Cache()    // ~/.cache/core on Linux
//	paths.Runtime()  // /run/user/{uid}/core on Linux
type XDGPaths struct {
	prefix string // Application name (default: "core")
}

// XDG returns a platform-aware path resolver using the default "core" prefix.
//
//	configDir := config.XDG().Config()   // ~/.config/core on Linux
func XDG() *XDGPaths {
	return &XDGPaths{prefix: "core"}
}

// XDGWithPrefix returns a path resolver using a custom application prefix.
//
//	paths := config.XDGWithPrefix("myapp")
//	paths.Config()  // ~/.config/myapp on Linux
func XDGWithPrefix(prefix string) *XDGPaths {
	if prefix == "" {
		prefix = "core"
	}
	return &XDGPaths{prefix: prefix}
}

// Config returns the configuration directory suffixed with the prefix.
//
//	path := config.XDG().Config()  // ~/.config/core
func (x *XDGPaths) Config() string {
	return filepath.Join(xdgOrDefault("XDG_CONFIG_HOME", defaultConfigHome()), x.prefix)
}

// Data returns the persistent data directory suffixed with the prefix.
//
//	path := config.XDG().Data()  // ~/.local/share/core
func (x *XDGPaths) Data() string {
	return filepath.Join(xdgOrDefault("XDG_DATA_HOME", defaultDataHome()), x.prefix)
}

// Cache returns the cache directory (safe to delete) suffixed with the prefix.
//
//	path := config.XDG().Cache()  // ~/.cache/core
func (x *XDGPaths) Cache() string {
	return filepath.Join(xdgOrDefault("XDG_CACHE_HOME", defaultCacheHome()), x.prefix)
}

// Runtime returns the ephemeral runtime directory suffixed with the prefix.
//
//	path := config.XDG().Runtime()  // /run/user/1000/core on Linux
func (x *XDGPaths) Runtime() string {
	return filepath.Join(xdgOrDefault("XDG_RUNTIME_DIR", defaultRuntimeDir()), x.prefix)
}

// Prefix returns the application prefix used by this resolver.
//
//	core.XDG().Prefix()  // "core"
func (x *XDGPaths) Prefix() string {
	return x.prefix
}

func xdgOrDefault(envVar, fallback string) string {
	if v := os.Getenv(envVar); v != "" {
		return v
	}
	return fallback
}

func home() string {
	if h, err := os.UserHomeDir(); err == nil {
		return h
	}
	return "."
}

func defaultConfigHome() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home(), "Library", "Application Support")
	case "windows":
		if v := os.Getenv("APPDATA"); v != "" {
			return v
		}
		return filepath.Join(home(), "AppData", "Roaming")
	default:
		return filepath.Join(home(), ".config")
	}
}

func defaultDataHome() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home(), "Library", "Application Support")
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return v
		}
		return filepath.Join(home(), "AppData", "Local")
	default:
		return filepath.Join(home(), ".local", "share")
	}
}

func defaultCacheHome() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home(), "Library", "Caches")
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "cache")
		}
		return filepath.Join(home(), "AppData", "Local", "cache")
	default:
		return filepath.Join(home(), ".cache")
	}
}

func defaultRuntimeDir() string {
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home(), "Library", "Caches")
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "temp")
		}
		return filepath.Join(home(), "AppData", "Local", "temp")
	default:
		return filepath.Join("/run/user", strconv.Itoa(os.Getuid()))
	}
}
