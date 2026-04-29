package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func ExampleSetConclaveRootFunc() {
	SetConclaveRootFunc(func(name string) core.Result {
		return core.Ok(core.PathJoin("/", "conclaves", name))
	})
	defer SetConclaveRootFunc(nil)
	result := conclaveRoot("alpha")
	root, _ := core.Cast[string](result)
	core.Println(result.OK, root)
	// Output: true /conclaves/alpha
}

func ExampleForConclave() {
	m := coreio.NewMockMedium()
	root := core.PathJoin("/", "conclaves", "alpha")
	_ = m.EnsureDir(core.PathJoin(root, ".core"))
	_ = m.Write(core.PathJoin(root, ".core", FileConfig), "app:\n  name: alpha\n")
	SetConclaveRootFunc(func(string) core.Result {
		return core.Ok(root)
	})
	defer SetConclaveRootFunc(nil)

	result := ForConclave("alpha", WithMedium(m))
	cfg, _ := core.Cast[*Config](result)
	var name string
	_ = cfg.Get("app.name", &name)
	core.Println(result.OK, name)
	// Output: true alpha
}
