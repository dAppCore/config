package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func exampleDiscoveryMedium() (*coreio.MockMedium, string) {
	m := coreio.NewMockMedium()
	root := core.PathJoin("/", "workspace", "repo")
	child := core.PathJoin(root, "cmd", "app")
	_ = m.EnsureDir(core.PathJoin(root, ".core"))
	_ = m.EnsureDir(core.PathJoin(root, ".git"))
	_ = m.EnsureDir(child)
	_ = m.Write(core.PathJoin(root, ".core", FileConfig), "app:\n  name: discovered\n")
	_ = m.Write(core.PathJoin(root, ".core", FileBuild), "version: 1\nproject:\n  name: app\n")
	return m, child
}

func ExampleDiscover() {
	cfg, err := Discover(WithMedium(coreio.NewMockMedium()))
	core.Println(err == nil && cfg != nil)
	// Output: true
}

func ExampleDiscoverFrom() {
	m, child := exampleDiscoveryMedium()
	cfg, err := DiscoverFrom(child, WithMedium(m))
	var name string
	_ = cfg.Get("app.name", &name)
	core.Println(err == nil, name)
	// Output: true discovered
}

func ExampleCoreDirs() {
	m, child := exampleDiscoveryMedium()
	dirs := CoreDirs(m, child)
	core.Println(core.PathBase(dirs[0]))
	// Output: .core
}

func ExampleFindManifest() {
	m, child := exampleDiscoveryMedium()
	path := FindManifest(m, child, FileBuild)
	core.Println(core.PathBase(path))
	// Output: build.yaml
}
