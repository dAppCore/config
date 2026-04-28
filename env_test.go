package config

import core "dappco.re/go"

func TestEnv_Env_Good(t *core.T) {
	t.Setenv("CORE_CONFIG_APP_NAME", "core")
	t.Setenv("CORE_CONFIG_DEV_EDITOR", "vim")

	got := map[string]string{}
	for key, value := range Env("CORE_CONFIG_") {
		got[key] = value.(string)
	}

	core.AssertEqual(t, map[string]string{
		"app.name":   "core",
		"dev.editor": "vim",
	}, got)
}

func TestEnv_LoadEnv_Bad(t *core.T) {
	t.Setenv("MYAPP_FEATURE_FLAG", "true")

	got := LoadEnv("CORE_CONFIG")
	core.AssertEmpty(t, got)
}

func TestEnv_Env_Ugly(t *core.T) {
	// Prefix normalisation accepts the trailing underscore form too.
	t.Setenv("MYAPP_FOO_BAR", "baz")

	var keys []string
	for key := range Env("MYAPP_") {
		keys = append(keys, key)
	}

	core.AssertEqual(t, []string{"foo.bar"}, keys)
}

func TestEnv_normaliseEnvPrefix_Good(t *core.T) {
	got := normaliseEnvPrefix("CORE_CONFIG")
	core.AssertEqual(t, "CORE_CONFIG_", got)

	got = normaliseEnvPrefix("CORE_CONFIG_")
	core.AssertEqual(t, "CORE_CONFIG_", got)
}

func TestEnv_normaliseEnvPrefix_Bad(t *core.T) {
	got := normaliseEnvPrefix("")
	core.AssertEqual(t, "", got)
	core.AssertFalse(t, core.HasSuffix(got, "_"))
}

func TestEnv_normaliseEnvPrefix_Ugly(t *core.T) {
	// A nil-like value is still normalised consistently.
	got := normaliseEnvPrefix("my_app")
	core.AssertEqual(t, "my_app_", got)
	core.AssertTrue(t, core.HasSuffix(got, "_"))
}

func TestEnv_Env_Bad(t *core.T) {
	t.Setenv("AX7_OTHER_NAME", "codex")
	got := map[string]any{}
	for key, value := range Env("AX7_CONFIG") {
		got[key] = value
	}
	core.AssertEmpty(t, got)
}

func TestEnv_LoadEnv_Good(t *core.T) {
	t.Setenv("AX7_LOAD_NAME", "codex")
	got := LoadEnv("AX7_LOAD")
	core.AssertEqual(t, "codex", got["name"])
}

func TestEnv_LoadEnv_Ugly(t *core.T) {
	t.Setenv("AX7_EMPTY_VALUE", "")
	got := LoadEnv("AX7_EMPTY")
	core.AssertEqual(t, "", got["value"])
}
