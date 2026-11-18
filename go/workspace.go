package config

import (
	core "github.com/dappcore/go"
	coreio "github.com/dappcore/go-io/go"
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
