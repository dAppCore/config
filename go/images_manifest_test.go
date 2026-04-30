package config

import (
	core "dappco.re/go"
	"io/fs"
	"time"

	coreio "dappco.re/go/io"
)

const imagesManifestCoreDev = "core-dev"

type failingImagesWriteMedium struct {
	*coreio.MockMedium
}

func withDefaultImagesManifestMedium(t *core.T, medium coreio.Medium) {
	t.Helper()
	previous := defaultImagesManifestMedium
	defaultImagesManifestMedium = func() coreio.Medium {
		return medium
	}
	t.Cleanup(func() {
		defaultImagesManifestMedium = previous
	})
}

func (m failingImagesWriteMedium) WriteMode(string, string, fs.FileMode) error {
	return core.NewError("write failed")
}

func TestImagesManifest_SaveImagesManifest_LoadSave_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)

	manifest := &ImagesManifest{
		Images: map[string]ImageInfo{
			imagesManifestCoreDev: {
				Version:    "1.2.3",
				SHA256:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
				Downloaded: time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC),
				Source:     "github",
			},
		},
	}

	core.RequireNoError(t, resultError(SaveImagesManifest(m, path, manifest)))

	loaded, err := imagesManifestResult(LoadImagesManifest(m, path))
	core.RequireNoError(t, err)
	core.RequireTrue(t, loaded != nil)
	core.AssertLen(t, loaded.Images, 1)
	core.AssertEqual(t, manifest.Images[imagesManifestCoreDev], loaded.Images[imagesManifestCoreDev])
}

func TestImagesManifest_ResolveImagesManifest_Good(t *core.T) {
	manifest, err := imagesManifestResult(ResolveImagesManifest(coreio.NewMockMedium()))
	core.RequireNoError(t, err)
	core.RequireTrue(t, manifest != nil)
	core.AssertEmpty(t, manifest.Images)
}

func TestImagesManifest_LoadImagesManifest_Missing_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)

	manifest, err := imagesManifestResult(LoadImagesManifest(m, path))
	core.RequireNoError(t, err)
	core.RequireTrue(t, manifest != nil)
	core.AssertEmpty(t, manifest.Images)
}

func TestImagesManifest_LoadImagesManifest_NilMedium_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)
	withDefaultImagesManifestMedium(t, m)
	core.RequireNoError(t, m.Write(path, `{"images":{}}`))

	manifest, err := imagesManifestResult(LoadImagesManifest(nil, path))
	core.RequireNoError(t, err)
	core.RequireTrue(t, manifest != nil)
	core.AssertEmpty(t, manifest.Images)
}

func TestImagesManifest_SaveImagesManifest_Nil_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)

	core.RequireNoError(t, resultError(SaveImagesManifest(m, path, nil)))

	content, err := m.Read(path)
	core.RequireNoError(t, err)
	core.AssertEqual(t, `{"images":{}}`, content)

	info, err := m.Stat(path)
	core.RequireNoError(t, err)
	core.AssertEqual(t, fs.FileMode(0600), info.Mode())
}

func TestImagesManifest_SaveImagesManifest_NilMedium_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)
	withDefaultImagesManifestMedium(t, m)

	core.RequireNoError(t, resultError(SaveImagesManifest(nil, path, &ImagesManifest{})))

	content, err := m.Read(path)
	core.RequireNoError(t, err)
	core.AssertEqual(t, `{"images":{}}`, content)
}

func TestImagesManifest_SaveImagesManifest_Bad(t *core.T) {
	m := failingImagesWriteMedium{MockMedium: coreio.NewMockMedium()}
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)

	err := resultError(SaveImagesManifest(m, path, &ImagesManifest{}))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to write images manifest")
}

func TestImagesManifest_LoadImagesManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)
	core.RequireNoError(t, m.EnsureDir(core.PathDir(path)))
	core.RequireNoError(t, m.Write(path, "{not-json"))

	manifest, err := imagesManifestResult(LoadImagesManifest(m, path))
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "failed to parse images manifest")
}

func TestImagesManifest_LoadImagesManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)
	core.RequireNoError(t, m.EnsureDir(core.PathDir(path)))
	bad := map[string]any{
		"images": map[string]any{
			imagesManifestCoreDev: map[string]any{
				"version": 123,
			},
		},
	}

	payload := core.JSONMarshal(bad)
	core.RequireTrue(t, payload.OK)
	core.RequireNoError(t, m.Write(path, string(payload.Value.([]byte))))

	manifest, err := imagesManifestResult(LoadImagesManifest(m, path))
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "schema validation failed")
}

func TestImagesManifest_ResolveImagesManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	home := core.Env("DIR_HOME")
	path := core.PathJoin(home, Directory, DirectoryImages, FileImagesManifest)
	core.RequireNoError(t, m.EnsureDir(core.PathDir(path)))
	core.RequireNoError(t, m.Write(path, "{not-json"))

	manifest, err := imagesManifestResult(ResolveImagesManifest(m))
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
}

func TestImagesManifest_ResolveImagesManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	manifest, err := imagesManifestResult(ResolveImagesManifest(m))
	core.RequireNoError(t, err)
	core.AssertNotNil(t, manifest)
	core.AssertEmpty(t, manifest.Images)
}

func TestImagesManifest_LoadImagesManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)
	core.RequireNoError(t, m.EnsureDir(core.PathDir(path)))
	core.RequireNoError(t, m.Write(path, `{"images":{"core-dev":{"version":"1.0.0","downloaded":"2026-04-15T12:00:00Z","source":"github"}}}`))

	manifest, err := imagesManifestResult(LoadImagesManifest(m, path))
	core.RequireNoError(t, err)
	core.AssertEqual(t, "1.0.0", manifest.Images[imagesManifestCoreDev].Version)
}

func TestImagesManifest_SaveImagesManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)
	manifest := &ImagesManifest{Images: map[string]ImageInfo{imagesManifestCoreDev: {Version: "1.0.0", Downloaded: time.Date(2026, time.April, 15, 12, 0, 0, 0, time.UTC), Source: "github"}}}

	err := resultError(SaveImagesManifest(m, path, manifest))
	core.AssertNoError(t, err)
	core.AssertTrue(t, m.Exists(path))
}

func TestImagesManifest_SaveImagesManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	path := core.PathJoin("home", ".core", DirectoryImages, FileImagesManifest)
	err := resultError(SaveImagesManifest(m, path, &ImagesManifest{}))
	core.AssertNoError(t, err)
	core.AssertTrue(t, m.Exists(path))
}
