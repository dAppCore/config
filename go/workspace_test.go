package config

import core "dappco.re/go"

import coreio "dappco.re/go/io"

func TestWorkspace_FindWorkspaceRoot_Good(t *core.T) {
	m := coreio.NewMockMedium()
	root := core.PathJoin("/", "workspace", "root")
	child := core.PathJoin(root, "service")

	core.AssertNoError(t, m.EnsureDir(core.PathJoin(root, ".core")))
	core.AssertNoError(t, m.EnsureDir(child))
	core.AssertNoError(t, m.Write(core.PathJoin(root, ".core", FileWorkspace), "version: 1\n"))

	core.AssertEqual(t, root, FindWorkspaceRoot(m, child))
}

func TestWorkspace_FindWorkspaceRoot_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	start := core.PathJoin("/", "workspace", "none")
	got := FindWorkspaceRoot(m, start)
	core.AssertEmpty(t, got)
}

func TestWorkspace_FindWorkspaceRoot_Ugly(t *core.T) {
	m := falseExistsMedium{coreio.NewMockMedium()}
	start := core.PathJoin("/", "workspace", "repo", "file.go")
	got := FindWorkspaceRoot(m, start)
	core.AssertEqual(t, "", got)
}
