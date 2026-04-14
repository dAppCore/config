package config

import (
	"path/filepath"
	"testing"

	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

func TestConclave_ForConclave_Good(t *testing.T) {
	tmp := t.TempDir()
	SetConclaveRootFunc(func(name string) (string, error) {
		return filepath.Join(tmp, name), nil
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	root := filepath.Join(tmp, "alpha", ".core")
	assert.NoError(t, coreio.Local.EnsureDir(root))
	assert.NoError(t, coreio.Local.Write(filepath.Join(root, "config.yaml"), "theme: dark\n"))

	cfg, err := ForConclave("alpha", WithMedium(coreio.Local))
	assert.NoError(t, err)

	var theme string
	assert.NoError(t, cfg.Get("theme", &theme))
	assert.Equal(t, "dark", theme)
}

func TestConclave_ForConclave_Bad(t *testing.T) {
	SetConclaveRootFunc(func(_ string) (string, error) {
		return "", assertResolverError()
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	_, err := ForConclave("missing")
	assert.Error(t, err)
}

func TestConclave_ForConclave_Ugly(t *testing.T) {
	// Nil resolver should fall back to the default — no panic.
	SetConclaveRootFunc(nil)
	cfg, err := ForConclave("test-conclave")
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
}

func TestConclave_SetConclaveRootFunc_Good(t *testing.T) {
	SetConclaveRootFunc(func(name string) (string, error) {
		return "/custom/" + name, nil
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	conclaveMu.RLock()
	resolver := conclaveRoot
	conclaveMu.RUnlock()

	root, err := resolver("a")
	assert.NoError(t, err)
	assert.Equal(t, "/custom/a", root)
}

func assertResolverError() error {
	return &assertErr{msg: "resolver failed"}
}

type assertErr struct{ msg string }

func (e *assertErr) Error() string { return e.msg }
