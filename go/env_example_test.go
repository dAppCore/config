package config

import core "dappco.re/go"

func ExampleEnv() {
	count := 0
	for range Env("CONFIG_EXAMPLE_NOT_SET_") {
		count++
	}
	core.Println(count)
	// Output: 0
}

func ExampleLoadEnv() {
	values := LoadEnv("CONFIG_EXAMPLE_NOT_SET_")
	core.Println(len(values))
	// Output: 0
}
