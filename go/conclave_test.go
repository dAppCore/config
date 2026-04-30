package config

import (
	core "dappco.re/go"
	"runtime"

	coreio "dappco.re/go/io"
)

func TestConclave_ForConclave_Good(t *core.T) {
	tmp := t.TempDir()
	SetConclaveRootFunc(func(name string) core.Result {
		return core.Ok(core.PathJoin(tmp, name))
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	root := core.PathJoin(tmp, "alpha", ".core")
	core.AssertNoError(t, coreio.Local.EnsureDir(root))
	core.AssertNoError(t, coreio.Local.Write(core.PathJoin(root, "config.yaml"), "theme: dark\n"))

	cfg := requireResultValue[*Config](t, ForConclave("alpha", WithMedium(coreio.Local)))

	var theme string
	core.AssertNoError(t, resultError(cfg.Get("theme", &theme)))
	core.AssertEqual(t, "dark", theme)
}

func TestConclave_ForConclave_Bad(t *core.T) {
	SetConclaveRootFunc(func(_ string) core.Result {
		return core.Fail(assertResolverError())
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	err := resultError(ForConclave("missing"))
	core.AssertError(t, err)
}

func TestConclave_ForConclave_Ugly(t *core.T) {
	// Nil resolver should fall back to the default — no panic.
	SetConclaveRootFunc(nil)
	cfg := requireResultValue[*Config](t, ForConclave("test-conclave"))
	core.AssertNotNil(t, cfg)
}

func TestConclave_ForConclave_InvalidName_Bad(t *core.T) {
	SetConclaveRootFunc(nil)
	err := resultError(ForConclave("../escape"))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid conclave name")
}

func TestConclave_ForConclave_SymlinkedCore_Bad(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	tmp := t.TempDir()
	conclaveDir := core.PathJoin(tmp, "conclave")
	realCore := core.PathJoin(tmp, "real-core")

	core.AssertNoError(t, coreio.Local.EnsureDir(conclaveDir))
	core.AssertNoError(t, coreio.Local.EnsureDir(realCore))
	core.AssertNoError(t, coreio.Local.Write(core.PathJoin(realCore, "config.yaml"), "theme: dark\n"))
	testSymlink(t, realCore, core.PathJoin(conclaveDir, ".core"))

	SetConclaveRootFunc(func(_ string) core.Result {
		return core.Ok(conclaveDir)
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	err := resultError(ForConclave("alpha"))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "symlinked conclave .core directory rejected")
}

func TestConclave_ForConclave_InheritsProject_Good(t *core.T) {
	// A Conclave inherits gaps from the project .core/ directory walked up
	// from the current working directory. The Conclave's own .core/ still
	// wins for keys it declares.
	projectDir := t.TempDir()
	conclaveDir := t.TempDir()

	core.AssertNoError(t, coreio.Local.EnsureDir(core.PathJoin(projectDir, ".core")))
	core.AssertNoError(t, coreio.Local.EnsureDir(core.PathJoin(projectDir, ".git")))
	core.AssertNoError(t, coreio.Local.Write(
		core.PathJoin(projectDir, ".core", "config.yaml"),
		"dev:\n  editor: vim\napp:\n  name: project\n",
	))

	core.AssertNoError(t, coreio.Local.EnsureDir(core.PathJoin(conclaveDir, ".core")))
	core.AssertNoError(t, coreio.Local.Write(
		core.PathJoin(conclaveDir, ".core", "config.yaml"),
		"app:\n  name: conclave\n",
	))

	SetConclaveRootFunc(func(_ string) core.Result {
		return core.Ok(conclaveDir)
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	// Switch cwd so Discover picks up the project layer.
	prev := testGetwd(t)
	testChdir(t, projectDir)
	t.Cleanup(func() { testChdir(t, prev) })

	cfg := requireResultValue[*Config](t, ForConclave("alpha", WithMedium(coreio.Local)))

	// Conclave wins on app.name.
	var name string
	core.AssertNoError(t, resultError(cfg.Get("app.name", &name)))
	core.AssertEqual(t, "conclave", name)

	// Project fills the gap on dev.editor.
	var editor string
	core.AssertNoError(t, resultError(cfg.Get("dev.editor", &editor)))
	core.AssertEqual(t, "vim", editor)
}

func TestConclave_SetConclaveRootFunc_Good(t *core.T) {
	SetConclaveRootFunc(func(name string) core.Result {
		return core.Ok("/custom/" + name)
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	conclaveMu.RLock()
	resolver := conclaveRoot
	conclaveMu.RUnlock()

	root := requireResultValue[string](t, resolver("a"))
	core.AssertEqual(t, "/custom/a", root)
}

func TestConclave_ForConclave_Isolation_Good(t *core.T) {
	// RFC §12.3: "Writes are isolated to the Conclave's .core/ directory.
	// alpha.Set("theme", "dark"), beta.Get("theme", &t) // unchanged"
	//
	// Two conclaves under different roots must not share state: a Set in
	// alpha is invisible to beta, and each Commit writes only to its own
	// .core/config.yaml.
	tmp := t.TempDir()
	SetConclaveRootFunc(func(name string) core.Result {
		return core.Ok(core.PathJoin(tmp, name))
	})
	t.Cleanup(func() { SetConclaveRootFunc(nil) })

	alpha := requireResultValue[*Config](t, ForConclave("workspace-alpha", WithMedium(coreio.Local)))
	beta := requireResultValue[*Config](t, ForConclave("workspace-beta", WithMedium(coreio.Local)))

	core.AssertNoError(t, resultError(alpha.Set("theme", "dark")))
	core.AssertNoError(t, resultError(alpha.Commit()))

	// beta was created before alpha's Set — its in-memory view is untouched.
	var betaTheme string
	err := resultError(beta.Get("theme", &betaTheme))
	core.AssertError(t, err)

	// Alpha's on-disk config contains theme; beta's root has no config file yet.
	alphaFile := core.PathJoin(tmp, "workspace-alpha", ".core", "config.yaml")
	betaFile := core.PathJoin(tmp, "workspace-beta", ".core", "config.yaml")

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
	r := resolver("alpha")
	core.AssertNoError(t, resultError(r))
	got := resultValue[string](r)
	core.AssertContains(t, got, core.PathJoin("conclaves", "alpha"))
}

func TestConclave_SetConclaveRootFunc_Ugly(t *core.T) {
	want := core.NewError("resolver refused")
	SetConclaveRootFunc(func(string) core.Result { return core.Fail(want) })
	t.Cleanup(func() { SetConclaveRootFunc(nil) })
	conclaveMu.RLock()
	resolver := conclaveRoot
	conclaveMu.RUnlock()
	r := resolver("alpha")
	got := resultValue[string](r)
	core.AssertEqual(t, "", got)
	core.AssertErrorIs(t, resultError(r), want)
}
