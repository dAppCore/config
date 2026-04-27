package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSchema_ValidateSchema_Good(t *testing.T) {
	raw := map[string]any{
		"version": 1,
		"app": map[string]any{
			"name":    "core",
			"version": "0.1.0",
		},
	}

	assert.NoError(t, validateSchema("/tmp/.core/config.yaml", raw))
}

func TestSchema_ValidateSchema_Bad(t *testing.T) {
	raw := map[string]any{
		"targets": 42,
	}

	err := validateSchema("/tmp/.core/build.yaml", raw)
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	assert.Contains(t, err.Error(), "schema validation failed")
}

func TestSchema_ValidateSchema_Ugly(t *testing.T) {
	assert.NoError(t, validateSchema("/tmp/.core/notes.txt", map[string]any{"anything": "goes"}))
	assert.NoError(t, validateSchema("/tmp/.core/config.yaml", map[string]any{}))
}
