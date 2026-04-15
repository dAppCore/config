package config

import (
	"encoding/json"
	"errors"
	"io/fs"
	"strings"
	"time"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"github.com/xeipuuv/gojsonschema"
)

// ImagesManifest tracks user-level LinuxKit image metadata under
// ~/.core/images/manifest.json.
//
//	manifest, err := config.ResolveImagesManifest(io.Local)
type ImagesManifest struct {
	Images map[string]ImageInfo `json:"images" yaml:"images"`
}

// ImageInfo stores metadata for a single installed image.
//
//	info := config.ImageInfo{
//	    Version:    "1.0.0",
//	    SHA256:     "abc123",
//	    Downloaded:  time.Now(),
//	    Source:     "github",
//	}
type ImageInfo struct {
	Version    string    `json:"version" yaml:"version"`
	SHA256     string    `json:"sha256,omitempty" yaml:"sha256,omitempty"`
	Downloaded time.Time `json:"downloaded" yaml:"downloaded"`
	Source     string    `json:"source" yaml:"source"`
}

// ResolveImagesManifest loads the user-global images registry from
// ~/.core/images/manifest.json. If the file does not exist, it returns an
// empty manifest instead of an error so callers can initialise the registry
// on demand.
func ResolveImagesManifest(medium coreio.Medium) (*ImagesManifest, error) {
	if medium == nil {
		medium = coreio.Local
	}
	path := FindUserImagesManifest(medium)
	if path == "" {
		return &ImagesManifest{Images: map[string]ImageInfo{}}, nil
	}
	return LoadImagesManifest(medium, path)
}

// LoadImagesManifest reads a JSON images registry from disk.
func LoadImagesManifest(medium coreio.Medium, path string) (*ImagesManifest, error) {
	if medium == nil {
		medium = coreio.Local
	}
	manifest := &ImagesManifest{Images: map[string]ImageInfo{}}

	content, err := medium.Read(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return manifest, nil
		}
		return nil, coreerr.E("config.LoadImagesManifest", "failed to read images manifest: "+path, err)
	}

	var raw map[string]any
	if err := json.Unmarshal([]byte(content), &raw); err != nil {
		return nil, coreerr.E("config.LoadImagesManifest", "failed to parse images manifest: "+path, err)
	}
	if err := validateImagesSchema(path, raw); err != nil {
		return nil, err
	}
	if err := json.Unmarshal([]byte(content), manifest); err != nil {
		return nil, coreerr.E("config.LoadImagesManifest", "failed to decode images manifest: "+path, err)
	}
	if manifest.Images == nil {
		manifest.Images = map[string]ImageInfo{}
	}
	return manifest, nil
}

// SaveImagesManifest writes the JSON images registry to disk.
func SaveImagesManifest(medium coreio.Medium, path string, manifest *ImagesManifest) error {
	if medium == nil {
		medium = coreio.Local
	}
	if manifest == nil {
		manifest = &ImagesManifest{Images: map[string]ImageInfo{}}
	}
	if manifest.Images == nil {
		manifest.Images = map[string]ImageInfo{}
	}

	payload, err := json.Marshal(manifest)
	if err != nil {
		return coreerr.E("config.SaveImagesManifest", "failed to marshal images manifest", err)
	}

	dir := core.PathDir(path)
	if err := medium.EnsureDir(dir); err != nil {
		return coreerr.E("config.SaveImagesManifest", "failed to create images manifest directory: "+dir, err)
	}
	if err := medium.WriteMode(path, string(payload), 0o600); err != nil {
		return coreerr.E("config.SaveImagesManifest", "failed to write images manifest: "+path, err)
	}
	return nil
}

func validateImagesSchema(path string, raw map[string]any) error {
	if len(raw) == 0 {
		return nil
	}

	schemaBody, err := schemaFS.ReadFile("schema/images.schema.json")
	if err != nil {
		return coreerr.E("config.validateImagesSchema", "failed to read embedded schema: schema/images.schema.json", err)
	}

	documentBody, err := json.Marshal(raw)
	if err != nil {
		return coreerr.E("config.validateImagesSchema", "failed to encode images manifest for schema validation: "+path, err)
	}

	result, err := gojsonschema.Validate(
		gojsonschema.NewBytesLoader(schemaBody),
		gojsonschema.NewBytesLoader(documentBody),
	)
	if err != nil {
		return coreerr.E("config.validateImagesSchema", "schema validation failed: "+path, err)
	}
	if result.Valid() {
		return nil
	}

	var problems []string
	for _, issue := range result.Errors() {
		problems = append(problems, issue.String())
	}
	return coreerr.E("config.validateImagesSchema", "schema validation failed: "+path+": "+strings.Join(problems, "; "), nil)
}
