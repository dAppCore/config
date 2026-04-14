package config

import (
	"path/filepath"
	"strings"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
)

// ForConclave returns a config scoped to the named Conclave.
//
// The function inherits merged project/global settings and applies Conclave-local
// overrides from <conclave-root>/.core/.
func ForConclave(name string) (*Config, error) {
	root := strings.TrimSpace(name)
	if root == "" {
		return nil, core.E("config.ForConclave", "conclave name is required", nil)
	}

	root = conclaveEnvRoot(root)
	if root == "" {
		return nil, core.E("config.ForConclave", "conclave root not resolved", nil)
	}

	parent, err := Discover()
	if err != nil {
		return nil, err
	}

	conclaveRoot := filepath.Join(root, ".core")
	conclaveCfg, err := discoverConfigForDir(coreio.Local, conclaveRoot, parent.env)
	if err != nil {
		return nil, err
	}
	conclaveCfg.path = filepath.Join(conclaveRoot, "config.yaml")

	// Conclave overrides parent config, parent fills remaining keys.
	conclaveCfg.MergeFrom(parent)
	return conclaveCfg, nil
}

func conclaveEnvRoot(name string) string {
	variant := strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	vars := []string{
		"CORE_CONCLAVE_" + variant + "_ROOT",
		"CONCLAVE_" + variant + "_ROOT",
		"CORE_CONCLAVE_ROOT",
		"CONCLAVE_ROOT",
	}
	for _, env := range vars {
		if value := core.Env(env); value != "" {
			return value
		}
	}
	return ""
}
