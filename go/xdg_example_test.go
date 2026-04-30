package config

import core "dappco.re/go"

func ExampleXDG() {
	paths := XDG()
	core.Println(paths.Prefix())
	// Output: core
}

func ExampleXDGWithPrefix() {
	paths := XDGWithPrefix("app")
	core.Println(paths.Prefix())
	// Output: app
}

func ExampleXDGPaths_Config() {
	paths := XDGWithPrefix("app")
	core.Println(core.HasSuffix(paths.Config(), core.PathJoin("app")))
	// Output: true
}

func ExampleXDGPaths_Data() {
	paths := XDGWithPrefix("app")
	core.Println(core.HasSuffix(paths.Data(), core.PathJoin("app")))
	// Output: true
}

func ExampleXDGPaths_Cache() {
	paths := XDGWithPrefix("app")
	core.Println(core.HasSuffix(paths.Cache(), core.PathJoin("app")))
	// Output: true
}

func ExampleXDGPaths_Runtime() {
	paths := XDGWithPrefix("app")
	core.Println(core.HasSuffix(paths.Runtime(), core.PathJoin("app")))
	// Output: true
}

func ExampleXDGPaths_Prefix() {
	paths := XDGWithPrefix("app")
	core.Println(paths.Prefix())
	// Output: app
}
