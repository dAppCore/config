package config

import core "dappco.re/go"

func TestSchema_ValidateSchema_Good(t *core.T) {
	raw := map[string]any{
		"version": 1,
		"app": map[string]any{
			"name":    "core",
			"version": "0.1.0",
		},
	}

	core.AssertNoError(t, validateSchema("/tmp/.core/config.yaml", raw))
}

func TestSchema_ValidateSchema_Bad(t *core.T) {
	raw := map[string]any{
		"targets": 42,
	}

	err := validateSchema("/tmp/.core/build.yaml", raw)
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	core.AssertContains(t, err.Error(), "schema validation failed")
}

func TestSchema_ValidateSchema_Ugly(t *core.T) {
	unknownErr := validateSchema("/tmp/.core/notes.txt", map[string]any{"anything": "goes"})
	emptyErr := validateSchema("/tmp/.core/config.yaml", map[string]any{})
	core.AssertNoError(t, unknownErr)
	core.AssertNoError(t, emptyErr)
}
