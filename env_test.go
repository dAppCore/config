package config_test

import (
	. "dappco.re/go"
	config "dappco.re/go/config"
)

func TestEnv_Env_Good(t *T) {
	t.Setenv("AX_CONFIG_FOO_BAR", "baz")
	t.Setenv("AX_CONFIG_ALPHA", "first")

	var keys []string
	var values []any
	for key, value := range config.Env("AX_CONFIG_") {
		keys = append(keys, key)
		values = append(values, value)
	}

	AssertEqual(t, []string{"alpha", "foo.bar"}, keys)
	AssertEqual(t, []any{"first", "baz"}, values)
}

func TestEnv_Env_Bad(t *T) {
	t.Setenv("AX_CONFIG_FOO", "bar")

	var keys []string
	for key := range config.Env("OTHER_CONFIG_") {
		keys = append(keys, key)
	}

	AssertEmpty(t, keys)
}

func TestEnv_Env_Ugly(t *T) {
	t.Setenv("AX_CONFIG_FOO_BAR", "baz")

	var keys []string
	for key := range config.Env("AX_CONFIG") {
		keys = append(keys, key)
	}

	AssertEqual(t, []string{"foo.bar"}, keys)
}

func TestEnv_LoadEnv_Good(t *T) {
	t.Setenv("AX_CONFIG_FOO_BAR", "baz")

	data := config.LoadEnv("AX_CONFIG_")

	AssertLen(t, data, 1)
	AssertEqual(t, "baz", data["foo.bar"])
}

func TestEnv_LoadEnv_Bad(t *T) {
	t.Setenv("AX_CONFIG_FOO_BAR", "baz")

	data := config.LoadEnv("OTHER_CONFIG_")

	AssertEmpty(t, data)
}

func TestEnv_LoadEnv_Ugly(t *T) {
	t.Setenv("AX_CONFIG_FOO_BAR", "baz")

	data := config.LoadEnv("AX_CONFIG")

	AssertLen(t, data, 1)
	AssertEqual(t, "baz", data["foo.bar"])
}
