package config

import (
	"testing"

	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

func TestManifest_LoadManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/pkg/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\nlicence: EUPL-1.2\n"

	var pkg PackageManifest
	err := LoadManifest(m, "/pkg/.core/manifest.yaml", &pkg)
	assert.NoError(t, err)
	assert.Equal(t, "go-io", pkg.Code)
	assert.Equal(t, "Core I/O", pkg.Name)
	assert.Equal(t, "0.3.0", pkg.Version)
	assert.Equal(t, "EUPL-1.2", pkg.Licence)
}

func TestManifest_LoadManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	var pkg PackageManifest
	err := LoadManifest(m, "/nonexistent.yaml", &pkg)
	assert.Error(t, err)
}

func TestManifest_LoadManifest_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/bad.yaml"] = "this is: [not: valid: yaml"

	var pkg PackageManifest
	err := LoadManifest(m, "/bad.yaml", &pkg)
	assert.Error(t, err)
}

func TestManifest_LoadManifest_Build_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/build.yaml"] = "name: core\noutput: dist\ncgo: false\ntargets:\n  - os: linux\n    arch: amd64\n  - os: darwin\n    arch: arm64\n"

	var build BuildManifest
	err := LoadManifest(m, "/.core/build.yaml", &build)
	assert.NoError(t, err)
	assert.Equal(t, "core", build.Name)
	assert.Equal(t, "dist", build.Output)
	assert.Len(t, build.Targets, 2)
	assert.Equal(t, "linux", build.Targets[0].OS)
	assert.Equal(t, "amd64", build.Targets[0].Arch)
}

func TestManifest_LoadManifest_View_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/view.yaml"] = "code: photo-browser\nname: Photo Browser\npermissions:\n  clipboard: true\n  filesystem: true\n"

	var view ViewManifest
	err := LoadManifest(m, "/.core/view.yaml", &view)
	assert.NoError(t, err)
	assert.Equal(t, "photo-browser", view.Code)
	assert.True(t, view.Permissions.Clipboard)
	assert.True(t, view.Permissions.Filesystem)
}
