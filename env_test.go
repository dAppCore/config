package config_test

import (
	. "dappco.re/go"
	config "dappco.re/go/config"
)

func TestEnv_Env_Good(t *T) {
	t.Setenv(testFooBarEnv, testFooBarValue)
	t.Setenv("AX_CONFIG_ALPHA", "first")

	var keys []string
	var values []any
	for key, value := range config.Env(testAXConfigPrefixWithSeparator) {
		keys = append(keys, key)
		values = append(values, value)
	}

	AssertEqual(t, []string{"alpha", testFooBarKey}, keys)
	AssertEqual(t, []any{"first", testFooBarValue}, values)
}

func TestEnv_Env_Bad(t *T) {
	t.Setenv("AX_CONFIG_FOO", "bar")

	var keys []string
	for key := range config.Env(testOtherConfigPrefix) {
		keys = append(keys, key)
	}

	AssertEmpty(t, keys)
}

func TestEnv_Env_Ugly(t *T) {
	t.Setenv(testFooBarEnv, testFooBarValue)

	var keys []string
	for key := range config.Env(testAXConfigPrefix) {
		keys = append(keys, key)
	}

	AssertEqual(t, []string{testFooBarKey}, keys)
}

func TestEnv_LoadEnv_Good(t *T) {
	t.Setenv(testFooBarEnv, testFooBarValue)

	data := config.LoadEnv(testAXConfigPrefixWithSeparator)

	AssertLen(t, data, 1)
	AssertEqual(t, testFooBarValue, data[testFooBarKey])
}

func TestEnv_LoadEnv_Bad(t *T) {
	t.Setenv(testFooBarEnv, testFooBarValue)

	data := config.LoadEnv(testOtherConfigPrefix)

	AssertEmpty(t, data)
}

func TestEnv_LoadEnv_Ugly(t *T) {
	t.Setenv(testFooBarEnv, testFooBarValue)

	data := config.LoadEnv(testAXConfigPrefix)

	AssertLen(t, data, 1)
	AssertEqual(t, testFooBarValue, data[testFooBarKey])
}
