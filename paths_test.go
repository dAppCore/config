package config

import (
	"io/fs"
	"testing"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type symlinkMockMedium struct {
	*coreio.MockMedium
	symlinks map[string]bool
}

func (m symlinkMockMedium) IsSymlink(path string) bool {
	return m.symlinks[path]
}

func TestPaths_IsSafePathElement_Good(t *testing.T) {
	for _, part := range []string{
		"repo",
		"config.yaml",
		"core-dev",
		"manifest_1",
	} {
		t.Run(part, func(t *testing.T) {
			assert.True(t, isSafePathElement(part))
		})
	}
}

func TestPaths_IsSafePathElement_Bad(t *testing.T) {
	for _, part := range []string{
		"",
		".",
		"..",
	} {
		t.Run(part, func(t *testing.T) {
			assert.False(t, isSafePathElement(part))
		})
	}
}

func TestPaths_IsSafePathElement_Ugly(t *testing.T) {
	for _, part := range []string{
		"./repo",
		"repo/../repo",
		"repo//service",
		"repo/./service",
		"repo\\service",
	} {
		t.Run(part, func(t *testing.T) {
			assert.False(t, isSafePathElement(part))
		})
	}
}

func TestPaths_IsSymlinkedCoreDir_Good(t *testing.T) {
	coreDir := core.Path("repo", ".core")
	m := symlinkMockMedium{
		MockMedium: coreio.NewMockMedium(),
		symlinks:   map[string]bool{coreDir: true},
	}
	require.NoError(t, m.EnsureDir(coreDir))

	assert.True(t, isSymlinkedCoreDir(m, coreDir))
}

func TestPaths_IsSymlinkedCoreDir_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	coreDir := core.Path("repo", ".core")

	require.NoError(t, m.EnsureDir(coreDir))

	assert.False(t, isSymlinkedCoreDir(m, coreDir))
}

func TestPaths_IsSymlinkedCoreDir_Ugly(t *testing.T) {
	previous := localLstat
	localLstat = func(string) (fs.FileInfo, error) {
		return coreio.NewFileInfo(".core", 0, fs.ModeSymlink, time.Now(), false), nil
	}
	t.Cleanup(func() {
		localLstat = previous
	})

	assert.True(t, isSymlinkedCoreDir(coreio.Local, core.Path("local", ".core")))
	assert.False(t, isSymlinkedCoreDir(coreio.NewMockMedium(), core.Path("local", ".core")))
}
