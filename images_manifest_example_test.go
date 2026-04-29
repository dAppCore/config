package config

import (
	"time"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
)

func exampleImagesManifest() *ImagesManifest {
	return &ImagesManifest{Images: map[string]ImageInfo{
		"core-dev": {
			Version:    "1.0.0",
			SHA256:     "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			Downloaded: time.Unix(0, 0).UTC(),
			Source:     "example",
		},
	}}
}

func ExampleResolveImagesManifest() {
	m := coreio.NewMockMedium()
	manifest, err := ResolveImagesManifest(m)
	core.Println(err == nil, len(manifest.Images))
	// Output: true 0
}

func ExampleLoadImagesManifest() {
	m := coreio.NewMockMedium()
	path := core.PathJoin("/", "home", ".core", DirectoryImages, FileImagesManifest)
	_ = SaveImagesManifest(m, path, exampleImagesManifest())
	manifest, err := LoadImagesManifest(m, path)
	core.Println(err == nil, manifest.Images["core-dev"].Version)
	// Output: true 1.0.0
}

func ExampleSaveImagesManifest() {
	m := coreio.NewMockMedium()
	path := core.PathJoin("/", "home", ".core", DirectoryImages, FileImagesManifest)
	err := SaveImagesManifest(m, path, exampleImagesManifest())
	core.Println(err == nil && m.Exists(path))
	// Output: true
}
