package config

import (
	core "dappco.re/go"
	"errors"
	"os"
	"path/filepath"
	"runtime"

	coreio "dappco.re/go/io"
)

// osGetwd / osChdir wrap os.Getwd and os.Chdir so test helpers can stay
// explicit about their side-effects without spreading raw os calls around.
func osGetwd() (string, error) { return os.Getwd() }
func osChdir(dir string) error { return os.Chdir(dir) }

func TestConclave_ForConclave_Good(t *core.T) {
	tmp := t.TempDir()
	SetConclaveRootFunc(func(name string) (string, error) {
		return filepath.Join(tmp, name), nil
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	root := filepath.Join(tmp, "alpha", ".core")
	core.AssertNoError(t, coreio.Local.EnsureDir(root))
	core.AssertNoError(t, coreio.Local.Write(filepath.Join(root, "config.yaml"), "theme: dark\n"))

	cfg, err := ForConclave("alpha", WithMedium(coreio.Local))
	core.AssertNoError(t, err)

	var theme string
	core.AssertNoError(t, cfg.Get("theme", &theme))
	core.AssertEqual(t, "dark", theme)
}

func TestConclave_ForConclave_Bad(t *core.T) {
	SetConclaveRootFunc(func(_ string) (string, error) {
		return "", assertResolverError()
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	_, err := ForConclave("missing")
	core.AssertError(t, err)
}

func TestConclave_ForConclave_Ugly(t *core.T) {
	// Nil resolver should fall back to the default — no panic.
	SetConclaveRootFunc(nil)
	cfg, err := ForConclave("test-conclave")
	core.AssertNoError(t, err)
	core.AssertNotNil(t, cfg)
}

func TestConclave_ForConclave_InvalidName_Bad(t *core.T) {
	SetConclaveRootFunc(nil)
	_, err := ForConclave("../escape")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid conclave name")
}

func TestConclave_ForConclave_SymlinkedCore_Bad(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	tmp := t.TempDir()
	conclaveDir := filepath.Join(tmp, "conclave")
	realCore := filepath.Join(tmp, "real-core")

	core.AssertNoError(t, coreio.Local.EnsureDir(conclaveDir))
	core.AssertNoError(t, coreio.Local.EnsureDir(realCore))
	core.AssertNoError(t, coreio.Local.Write(filepath.Join(realCore, "config.yaml"), "theme: dark\n"))
	core.AssertNoError(t, os.Symlink(realCore, filepath.Join(conclaveDir, ".core")))

	SetConclaveRootFunc(func(_ string) (string, error) {
		return conclaveDir, nil
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	_, err := ForConclave("alpha")
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "symlinked conclave .core directory rejected")
}

func TestConclave_ForConclave_InheritsProject_Good(t *core.T) {
	// A Conclave inherits gaps from the project .core/ directory walked up
	// from the current working directory. The Conclave's own .core/ still
	// wins for keys it declares.
	projectDir := t.TempDir()
	conclaveDir := t.TempDir()

	core.AssertNoError(t, coreio.Local.EnsureDir(filepath.Join(projectDir, ".core")))
	core.AssertNoError(t, coreio.Local.EnsureDir(filepath.Join(projectDir, ".git")))
	core.AssertNoError(t, coreio.Local.Write(
		filepath.Join(projectDir, ".core", "config.yaml"),
		"dev:\n  editor: vim\napp:\n  name: project\n",
	))

	core.AssertNoError(t, coreio.Local.EnsureDir(filepath.Join(conclaveDir, ".core")))
	core.AssertNoError(t, coreio.Local.Write(
		filepath.Join(conclaveDir, ".core", "config.yaml"),
		"app:\n  name: conclave\n",
	))

	SetConclaveRootFunc(func(_ string) (string, error) {
		return conclaveDir, nil
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	// Switch cwd so Discover picks up the project layer.
	prev, err := osGetwd()
	core.AssertNoError(t, err)
	core.AssertNoError(t, osChdir(projectDir))
	t.Cleanup(func() { _ = osChdir(prev) })

	cfg, err := ForConclave("alpha", WithMedium(coreio.Local))
	core.AssertNoError(t, err)

	// Conclave wins on app.name.
	var name string
	core.AssertNoError(t, cfg.Get("app.name", &name))
	core.AssertEqual(t, "conclave", name)

	// Project fills the gap on dev.editor.
	var editor string
	core.AssertNoError(t, cfg.Get("dev.editor", &editor))
	core.AssertEqual(t, "vim", editor)
}

func TestConclave_SetConclaveRootFunc_Good(t *core.T) {
	SetConclaveRootFunc(func(name string) (string, error) {
		return "/custom/" + name, nil
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	conclaveMu.RLock()
	resolver := conclaveRoot
	conclaveMu.RUnlock()

	root, err := resolver("a")
	core.AssertNoError(t, err)
	core.AssertEqual(t, "/custom/a", root)
}

func TestConclave_Isolation_Good(t *core.T) {
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
	core.AssertNoError(t, err)
	beta, err := ForConclave("workspace-beta", WithMedium(coreio.Local))
	core.AssertNoError(t, err)

	core.AssertNoError(t, alpha.Set("theme", "dark"))
	core.AssertNoError(t, alpha.Commit())

	// beta was created before alpha's Set — its in-memory view is untouched.
	var betaTheme string
	err = beta.Get("theme", &betaTheme)
	core.AssertError(t, err)

	// Alpha's on-disk config contains theme; beta's root has no config file yet.
	alphaFile := filepath.Join(tmp, "workspace-alpha", ".core", "config.yaml")
	betaFile := filepath.Join(tmp, "workspace-beta", ".core", "config.yaml")

	body, err := coreio.Local.Read(alphaFile)
	core.AssertNoError(t, err)
	core.AssertContains(t, body, "theme")
	core.AssertContains(t, body, "dark")

	core.AssertFalse(t, coreio.Local.Exists(betaFile), "beta conclave must not have received alpha's write")
}

func assertResolverError() error {
	return &assertErr{msg: "resolver failed"}
}

type assertErr struct{ msg string }

func (e *assertErr) Error() string { return e.msg }

func TestConclave_SetConclaveRootFunc_Bad(t *core.T) {
	SetConclaveRootFunc(nil)
	t.Cleanup(func() { SetConclaveRootFunc(nil) })
	conclaveMu.RLock()
	resolver := conclaveRoot
	conclaveMu.RUnlock()
	got, err := resolver("alpha")
	core.AssertNoError(t, err)
	core.AssertContains(t, got, filepath.Join("conclaves", "alpha"))
}

func TestConclave_SetConclaveRootFunc_Ugly(t *core.T) {
	want := errors.New("resolver refused")
	SetConclaveRootFunc(func(string) (string, error) { return "", want })
	t.Cleanup(func() { SetConclaveRootFunc(nil) })
	conclaveMu.RLock()
	resolver := conclaveRoot
	conclaveMu.RUnlock()
	got, err := resolver("alpha")
	core.AssertEqual(t, "", got)
	core.AssertErrorIs(t, err, want)
}
