package config

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImagesManifest_LoadSave_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	path := filepath.Join(t.TempDir(), ".core", DirectoryImages, FileImagesManifest)

	manifest := &ImagesManifest{
		Images: map[string]ImageInfo{
			"core-dev": {
				Version:    "1.2.3",
				SHA256:     "abc123",
				Downloaded: time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC),
				Source:     "github",
			},
		},
	}

	require.NoError(t, SaveImagesManifest(m, path, manifest))

	loaded, err := LoadImagesManifest(m, path)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	require.Len(t, loaded.Images, 1)
	assert.Equal(t, manifest.Images["core-dev"], loaded.Images["core-dev"])
}

func TestImagesManifest_ResolveMissing_Good(t *testing.T) {
	manifest, err := ResolveImagesManifest(coreio.NewMockMedium())
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Empty(t, manifest.Images)
}

func TestImagesManifest_LoadImagesManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	path := filepath.Join(t.TempDir(), ".core", DirectoryImages, FileImagesManifest)
	require.NoError(t, m.EnsureDir(filepath.Dir(path)))
	require.NoError(t, m.Write(path, "{not-json"))

	manifest, err := LoadImagesManifest(m, path)
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse images manifest")
}

func TestImagesManifest_LoadImagesManifest_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	path := filepath.Join(t.TempDir(), ".core", DirectoryImages, FileImagesManifest)
	require.NoError(t, m.EnsureDir(filepath.Dir(path)))
	bad := map[string]any{
		"images": map[string]any{
			"core-dev": map[string]any{
				"version": 123,
			},
		},
	}

	payload, err := json.Marshal(bad)
	require.NoError(t, err)
	require.NoError(t, m.Write(path, string(payload)))

	manifest, err := LoadImagesManifest(m, path)
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation failed")
}
