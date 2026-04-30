package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

type symlinkReporter interface {
	IsSymlink(string) bool
}

var localLstat = func(path string) (core.FsFileInfo, error) {
	r := core.Lstat(path)
	if !r.OK {
		if err, ok := r.Value.(error); ok {
			return nil, err
		}
		return nil, core.NewError("lstat failed")
	}
	return r.Value.(core.FsFileInfo), nil
}

// isSafePathElement reports whether part is a single relative path element.
// It rejects absolute paths, traversal segments, and separators so public
// helpers cannot be tricked into leaving their intended directory roots.
func isSafePathElement(part string) bool {
	if part == "" {
		return false
	}
	if core.PathIsAbs(part) {
		return false
	}
	if core.Contains(part, "/") || core.Contains(part, `\`) {
		return false
	}
	clean := core.CleanPath(part, string(core.PathSeparator))
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

func isSymlinkedByLstat(lstat func(string) (core.FsFileInfo, error), path string) bool {
	info, err := lstat(path)
	if err != nil {
		return false
	}
	return info.Mode()&core.ModeSymlink != 0
}
