package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnv_Env_Good(t *testing.T) {
	t.Setenv("CORE_CONFIG_APP_NAME", "core")
	t.Setenv("CORE_CONFIG_DEV_EDITOR", "vim")

	got := map[string]string{}
	for key, value := range Env("CORE_CONFIG_") {
		got[key] = value.(string)
	}

	assert.Equal(t, map[string]string{
		"app.name":   "core",
		"dev.editor": "vim",
	}, got)
}

func TestEnv_LoadEnv_Bad(t *testing.T) {
	t.Setenv("MYAPP_FEATURE_FLAG", "true")

	assert.Empty(t, LoadEnv("CORE_CONFIG"))
}

func TestEnv_Env_Ugly(t *testing.T) {
	// Prefix normalisation accepts the trailing underscore form too.
	t.Setenv("MYAPP_FOO_BAR", "baz")

	var keys []string
	for key := range Env("MYAPP_") {
		keys = append(keys, key)
	}

	assert.Equal(t, []string{"foo.bar"}, keys)
}

func TestEnv_normaliseEnvPrefix_Good(t *testing.T) {
	got := normaliseEnvPrefix("CORE_CONFIG")
	assert.Equal(t, "CORE_CONFIG_", got)

	got = normaliseEnvPrefix("CORE_CONFIG_")
	assert.Equal(t, "CORE_CONFIG_", got)
}

func TestEnv_normaliseEnvPrefix_Bad(t *testing.T) {
	assert.Equal(t, "", normaliseEnvPrefix(""))
}

func TestEnv_normaliseEnvPrefix_Ugly(t *testing.T) {
	// A nil-like value is still normalised consistently.
	got := normaliseEnvPrefix("my_app")
	assert.Equal(t, "my_app_", got)
}
