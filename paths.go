package config

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	coreio "dappco.re/go/io"
)

type symlinkReporter interface {
	IsSymlink(string) bool
}

var localLstat = os.Lstat

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
	if reporter, ok := medium.(symlinkReporter); ok {
		return reporter.IsSymlink(path)
	}
	if medium != coreio.Local {
		return false
	}
	return isSymlinkedByLstat(localLstat, path)
}

// isSymlinkedLocalPath reports whether path is a symlink on the local
// filesystem. It is used for sensitive user-global registries that must not
// escape their expected on-disk roots via indirection.
func isSymlinkedLocalPath(path string) bool {
	return isSymlinkedByLstat(localLstat, path)
}

func isSymlinkedByLstat(lstat func(string) (fs.FileInfo, error), path string) bool {
	info, err := lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeSymlink != 0
}
