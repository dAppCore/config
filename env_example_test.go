package config_test

import (
	. "dappco.re/go"
	config "dappco.re/go/config"
)

func ExampleEnv() {
	Setenv("GO_CONFIG_EXAMPLE_HOST", "localhost")
	defer Unsetenv("GO_CONFIG_EXAMPLE_HOST")

	for key, value := range config.Env("GO_CONFIG_EXAMPLE") {
		Println(key, value)
	}
	// Output: host localhost
}

func ExampleLoadEnv() {
	Setenv("GO_CONFIG_EXAMPLE_HOST", "localhost")
	defer Unsetenv("GO_CONFIG_EXAMPLE_HOST")

	data := config.LoadEnv("GO_CONFIG_EXAMPLE")

	Println(data["host"])
	// Output: localhost
}
