package config

import (
	"path/filepath"
	"strings"
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
