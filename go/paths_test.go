package config

import (
	"io/fs"
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

type symlinkMockMedium struct {
	*coreio.MockMedium
	symlinks map[string]bool
}

func (m symlinkMockMedium) IsSymlink(path string) bool {
	return m.symlinks[path]
}

func TestPaths_isSafePathElement_Good(t *core.T) {
	for _, part := range []string{
		"repo",
		"config.yaml",
		"core-dev",
		"manifest_1",
	} {
		t.Run(part, func(t *core.T) {
			core.AssertTrue(t, isSafePathElement(part))
		})
	}
}

func TestPaths_isSafePathElement_Bad(t *core.T) {
	for _, part := range []string{
		"",
		".",
		"..",
	} {
		t.Run(part, func(t *core.T) {
			core.AssertFalse(t, isSafePathElement(part))
		})
	}
}

func TestPaths_isSafePathElement_Ugly(t *core.T) {
	for _, part := range []string{
		"./repo",
		"repo/../repo",
		"repo//service",
		"repo/./service",
		"repo\\service",
	} {
		t.Run(part, func(t *core.T) {
			core.AssertFalse(t, isSafePathElement(part))
		})
	}
}

func TestPaths_isSymlinkedCoreDir_Good(t *core.T) {
	coreDir := core.Path("repo", ".core")
	m := symlinkMockMedium{
		MockMedium: coreio.NewMockMedium(),
		symlinks:   map[string]bool{coreDir: true},
	}
	core.RequireNoError(t, m.EnsureDir(coreDir))

	core.AssertTrue(t, isSymlinkedCoreDir(m, coreDir))
}

func TestPaths_isSymlinkedCoreDir_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	coreDir := core.Path("repo", ".core")

	core.RequireNoError(t, m.EnsureDir(coreDir))

	core.AssertFalse(t, isSymlinkedCoreDir(m, coreDir))
}

func TestPaths_isSymlinkedCoreDir_Ugly(t *core.T) {
	previous := localLstat
	localLstat = func(string) (fs.FileInfo, error) {
		return coreio.NewFileInfo(".core", 0, fs.ModeSymlink, time.Now(), false), nil
	}
	t.Cleanup(func() {
		localLstat = previous
	})

	core.AssertTrue(t, isSymlinkedCoreDir(coreio.Local, core.Path("local", ".core")))
	core.AssertFalse(t, isSymlinkedCoreDir(coreio.NewMockMedium(), core.Path("local", ".core")))
}
