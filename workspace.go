package config

import (
	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

// FindWorkspaceManifest returns the nearest .core/workspace.yaml found while
// walking upward from start. It is a convenience wrapper around FindManifest.
//
//	path := config.FindWorkspaceManifest(io.Local, cwd)
func FindWorkspaceManifest(medium coreio.Medium, start string) string {
	return FindManifest(medium, start, FileWorkspace)
}

// ResolveWorkspaceManifest loads the nearest .core/workspace.yaml found while
// walking upward from start.
//
//	ws, err := config.ResolveWorkspaceManifest(io.Local, cwd)
//	if err != nil { /* no workspace or invalid YAML */ }
func ResolveWorkspaceManifest(medium coreio.Medium, start string) (*WorkspaceManifest, error) {
	if medium == nil {
		medium = coreio.Local
	}

	path := FindWorkspaceManifest(medium, start)
	if path == "" {
		return nil, coreerr.E("config.ResolveWorkspaceManifest", "no workspace manifest could be detected", nil)
	}

	var ws WorkspaceManifest
	if err := LoadManifest(medium, path, &ws); err != nil {
		return nil, err
	}
	return &ws, nil
}

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
