package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	coreio "dappco.re/go/io"
	"github.com/stretchr/testify/assert"
)

// osGetwd / osChdir wrap os.Getwd and os.Chdir so test helpers can stay
// explicit about their side-effects without spreading raw os calls around.
func osGetwd() (string, error) { return os.Getwd() }
func osChdir(dir string) error { return os.Chdir(dir) }

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

func TestConclave_ForConclave_InvalidName_Bad(t *testing.T) {
	SetConclaveRootFunc(nil)
	_, err := ForConclave("../escape")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid conclave name")
}

func TestConclave_ForConclave_SymlinkedCore_Bad(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	tmp := t.TempDir()
	conclaveDir := filepath.Join(tmp, "conclave")
	realCore := filepath.Join(tmp, "real-core")

	assert.NoError(t, coreio.Local.EnsureDir(conclaveDir))
	assert.NoError(t, coreio.Local.EnsureDir(realCore))
	assert.NoError(t, coreio.Local.Write(filepath.Join(realCore, "config.yaml"), "theme: dark\n"))
	assert.NoError(t, os.Symlink(realCore, filepath.Join(conclaveDir, ".core")))

	SetConclaveRootFunc(func(_ string) (string, error) {
		return conclaveDir, nil
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	_, err := ForConclave("alpha")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symlinked conclave .core directory rejected")
}

func TestConclave_ForConclave_InheritsProject_Good(t *testing.T) {
	// A Conclave inherits gaps from the project .core/ directory walked up
	// from the current working directory. The Conclave's own .core/ still
	// wins for keys it declares.
	projectDir := t.TempDir()
	conclaveDir := t.TempDir()

	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(projectDir, ".core")))
	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(projectDir, ".git")))
	assert.NoError(t, coreio.Local.Write(
		filepath.Join(projectDir, ".core", "config.yaml"),
		"dev:\n  editor: vim\napp:\n  name: project\n",
	))

	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(conclaveDir, ".core")))
	assert.NoError(t, coreio.Local.Write(
		filepath.Join(conclaveDir, ".core", "config.yaml"),
		"app:\n  name: conclave\n",
	))

	SetConclaveRootFunc(func(_ string) (string, error) {
		return conclaveDir, nil
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	// Switch cwd so Discover picks up the project layer.
	prev, err := osGetwd()
	assert.NoError(t, err)
	assert.NoError(t, osChdir(projectDir))
	t.Cleanup(func() { _ = osChdir(prev) })

	cfg, err := ForConclave("alpha", WithMedium(coreio.Local))
	assert.NoError(t, err)

	// Conclave wins on app.name.
	var name string
	assert.NoError(t, cfg.Get("app.name", &name))
	assert.Equal(t, "conclave", name)

	// Project fills the gap on dev.editor.
	var editor string
	assert.NoError(t, cfg.Get("dev.editor", &editor))
	assert.Equal(t, "vim", editor)
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

func TestConclave_Isolation_Good(t *testing.T) {
	// RFC §12.3: "Writes are isolated to the Conclave's .core/ directory.
	// alpha.Set("theme", "dark"), beta.Get("theme", &t) // unchanged"
	//
	// Two conclaves under different roots must not share state: a Set in
	// alpha is invisible to beta, and each Commit writes only to its own
	// .core/config.yaml.
	tmp := t.TempDir()
	SetConclaveRootFunc(func(name string) (string, error) {
		return filepath.Join(tmp, name), nil
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	alpha, err := ForConclave("workspace-alpha", WithMedium(coreio.Local))
	assert.NoError(t, err)
	beta, err := ForConclave("workspace-beta", WithMedium(coreio.Local))
	assert.NoError(t, err)

	assert.NoError(t, alpha.Set("theme", "dark"))
	assert.NoError(t, alpha.Commit())

	// beta was created before alpha's Set — its in-memory view is untouched.
	var betaTheme string
	err = beta.Get("theme", &betaTheme)
	assert.Error(t, err, "beta must not see alpha's writes")

	// Alpha's on-disk config contains theme; beta's root has no config file yet.
	alphaFile := filepath.Join(tmp, "workspace-alpha", ".core", "config.yaml")
	betaFile := filepath.Join(tmp, "workspace-beta", ".core", "config.yaml")

	body, err := coreio.Local.Read(alphaFile)
	assert.NoError(t, err)
	assert.Contains(t, body, "theme")
	assert.Contains(t, body, "dark")

	assert.False(t, coreio.Local.Exists(betaFile), "beta conclave must not have received alpha's write")
}

func assertResolverError() error {
	return &assertErr{msg: "resolver failed"}
}

type assertErr struct{ msg string }

func (e *assertErr) Error() string { return e.msg }
