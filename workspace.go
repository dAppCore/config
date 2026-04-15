package config

import (
	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

// FindWorkspaceRoot returns the directory that contains the nearest
// .core/workspace.yaml while walking upward from start. If no workspace
// manifest exists, it returns an empty string.
//
//	root := config.FindWorkspaceRoot(io.Local, cwd)
func FindWorkspaceRoot(medium coreio.Medium, start string) string {
	path := FindWorkspaceManifest(medium, start)
	if path == "" {
		return ""
	}
	return core.PathDir(core.PathDir(path))
}
