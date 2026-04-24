package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	coreio "dappco.re/go/io"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type failingImagesWriteMedium struct {
	*coreio.MockMedium
}

func (m failingImagesWriteMedium) WriteMode(string, string, fs.FileMode) error {
	return errors.New("write failed")
}

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

func TestImagesManifest_LoadImagesManifest_Missing_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	path := filepath.Join(t.TempDir(), ".core", DirectoryImages, FileImagesManifest)

	manifest, err := LoadImagesManifest(m, path)
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Empty(t, manifest.Images)
}

func TestImagesManifest_LoadImagesManifest_NilMedium_Good(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".core", DirectoryImages, FileImagesManifest)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(`{"images":{}}`), 0o600))

	manifest, err := LoadImagesManifest(nil, path)
	require.NoError(t, err)
	require.NotNil(t, manifest)
	assert.Empty(t, manifest.Images)
}

func TestImagesManifest_SaveImagesManifest_Nil_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	path := filepath.Join(t.TempDir(), ".core", DirectoryImages, FileImagesManifest)

	require.NoError(t, SaveImagesManifest(m, path, nil))

	content, err := m.Read(path)
	require.NoError(t, err)
	assert.JSONEq(t, `{"images":{}}`, content)

	info, err := m.Stat(path)
	require.NoError(t, err)
	assert.Equal(t, fs.FileMode(0600), info.Mode())
}

func TestImagesManifest_SaveImagesManifest_NilMedium_Good(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".core", DirectoryImages, FileImagesManifest)

	require.NoError(t, SaveImagesManifest(nil, path, &ImagesManifest{}))

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.JSONEq(t, `{"images":{}}`, string(content))
}

func TestImagesManifest_SaveImagesManifest_Bad(t *testing.T) {
	m := failingImagesWriteMedium{MockMedium: coreio.NewMockMedium()}
	path := filepath.Join(t.TempDir(), ".core", DirectoryImages, FileImagesManifest)

	err := SaveImagesManifest(m, path, &ImagesManifest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to write images manifest")
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
