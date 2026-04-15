package config

import (
	"os"
	"path/filepath"
	"strings"

	coreio "dappco.re/go/core/io"
)

// isSafePathElement reports whether part is a single relative path element.
// It rejects absolute paths, traversal segments, and separators so public
// helpers cannot be tricked into leaving their intended directory roots.
func isSafePathElement(part string) bool {
	if part == "" {
		return false
	}
	if filepath.IsAbs(part) {
		return false
	}
	if strings.ContainsAny(part, `/\`) {
		return false
	}
	clean := filepath.Clean(part)
	if clean != part {
		return false
	}
	switch clean {
	case ".", "..":
		return false
	default:
		return true
	}
}

// isSymlinkedCoreDir reports whether path points at a symlinked .core
// directory on the local filesystem. Discovery helpers use this to reject
// unsafe repository roots before they are traversed.
func isSymlinkedCoreDir(medium coreio.Medium, path string) bool {
	if medium != coreio.Local {
		return false
	}
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}
