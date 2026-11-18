package config

import (
	core "github.com/dappcore/go"
	coreio "github.com/dappcore/go-io/go"
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
