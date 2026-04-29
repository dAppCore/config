package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func ExampleSetConclaveRootFunc() {
	SetConclaveRootFunc(func(name string) (string, error) {
		return core.PathJoin("/", "conclaves", name), nil
	})
	defer SetConclaveRootFunc(nil)
	root, err := conclaveRoot("alpha")
	core.Println(err == nil, root)
	// Output: true /conclaves/alpha
}

func ExampleForConclave() {
	m := coreio.NewMockMedium()
	root := core.PathJoin("/", "conclaves", "alpha")
	_ = m.EnsureDir(core.PathJoin(root, ".core"))
	_ = m.Write(core.PathJoin(root, ".core", FileConfig), "app:\n  name: alpha\n")
	SetConclaveRootFunc(func(string) (string, error) {
		return root, nil
	})
	defer SetConclaveRootFunc(nil)

	cfg, err := ForConclave("alpha", WithMedium(m))
	var name string
	_ = cfg.Get("app.name", &name)
	core.Println(err == nil, name)
	// Output: true alpha
}
