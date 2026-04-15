package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	tmp := t.TempDir()
	realCore := filepath.Join(tmp, "real-core")
	linkCore := filepath.Join(tmp, ".core")

	require.NoError(t, os.MkdirAll(realCore, 0755))
	require.NoError(t, os.Symlink(realCore, linkCore))

	assert.True(t, isSymlinkedCoreDir(coreio.Local, linkCore))
}

func TestPaths_IsSymlinkedCoreDir_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	coreDir := filepath.Join(tmp, ".core")

	require.NoError(t, m.EnsureDir(coreDir))

	assert.False(t, isSymlinkedCoreDir(m, coreDir))
}

func TestPaths_IsSymlinkedCoreDir_Ugly(t *testing.T) {
	assert.False(t, isSymlinkedCoreDir(coreio.Local, filepath.Join(t.TempDir(), ".core")))
}
