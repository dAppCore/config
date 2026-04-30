package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func ExampleFindWorkspaceRoot() {
	m := coreio.NewMockMedium()
	root := core.PathJoin("/", "workspace", "repo")
	child := core.PathJoin(root, "service")
	_ = m.EnsureDir(core.PathJoin(root, ".core"))
	_ = m.EnsureDir(child)
	_ = m.Write(core.PathJoin(root, ".core", FileWorkspace), "version: 1\n")
	core.Println(FindWorkspaceRoot(m, child))
	// Output: /workspace/repo
}
