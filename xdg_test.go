package config

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestXdg_XDG_Good(t *testing.T) {
	paths := XDG()
	assert.Equal(t, "core", paths.Prefix())
	assert.True(t, strings.HasSuffix(paths.Config(), "/core") || strings.HasSuffix(paths.Config(), "\\core"))
	assert.True(t, strings.HasSuffix(paths.Data(), "/core") || strings.HasSuffix(paths.Data(), "\\core"))
	assert.True(t, strings.HasSuffix(paths.Cache(), "/core") || strings.HasSuffix(paths.Cache(), "\\core"))
	assert.True(t, strings.HasSuffix(paths.Runtime(), "/core") || strings.HasSuffix(paths.Runtime(), "\\core"))
}

func TestXdg_XDG_Bad(t *testing.T) {
	// An empty prefix falls back to the default "core" — no panic, no empty paths.
	paths := XDGWithPrefix("")
	assert.Equal(t, "core", paths.Prefix())
}

func TestXdg_XDG_Ugly(t *testing.T) {
	// Overriding XDG_CONFIG_HOME via env must change the resolved Config dir.
	t.Setenv("XDG_CONFIG_HOME", "/custom/config")
	paths := XDGWithPrefix("myapp")
	assert.Equal(t, "/custom/config/myapp", paths.Config())
}

func TestXdg_XDGWithPrefix_Good(t *testing.T) {
	paths := XDGWithPrefix("testing")
	assert.Equal(t, "testing", paths.Prefix())
	assert.Contains(t, paths.Config(), "testing")
}
