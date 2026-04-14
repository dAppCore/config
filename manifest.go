package config

import (
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"gopkg.in/yaml.v3"
)

// ViewManifest defines the structure of .core/view.yaml.
// Used by go-webview and dAppServer to configure window behaviour, permissions,
// and mounted slots.
//
//	var view config.ViewManifest
//	_ = config.LoadManifest(io.Local, ".core/view.yaml", &view)
type ViewManifest struct {
	Code        string          `yaml:"code"`
	Name        string          `yaml:"name"`
	Version     string          `yaml:"version"`
	Sign        string          `yaml:"sign"`
	Title       string          `yaml:"title"`
	Width       int             `yaml:"width"`
	Height      int             `yaml:"height"`
	Resizable   bool            `yaml:"resizable"`
	Layout      string          `yaml:"layout"`
	Slots       map[string]any  `yaml:"slots"`
	Modules     []string        `yaml:"modules"`
	Permissions ViewPermissions `yaml:"permissions"`
	Config      map[string]any  `yaml:"config"`
}

// ViewPermissions controls what a webview or application surface is allowed to do.
//
//	if view.Permissions.Clipboard { /* enable paste */ }
type ViewPermissions struct {
	Clipboard     bool     `yaml:"clipboard"`
	Filesystem    bool     `yaml:"filesystem"`
	Network       bool     `yaml:"network"`
	Notifications bool     `yaml:"notifications"`
	Camera        bool     `yaml:"camera"`
	Microphone    bool     `yaml:"microphone"`
	Read          []string `yaml:"read"`
	Net           []string `yaml:"net"`
	Run           []string `yaml:"run"`
}

// BuildManifest defines the structure of .core/build.yaml.
// Used by go-build to configure compilation targets and flags.
//
//	var build config.BuildManifest
//	_ = config.LoadManifest(io.Local, ".core/build.yaml", &build)
type BuildManifest struct {
	Name    string        `yaml:"name"`
	Output  string        `yaml:"output"`
	Targets []BuildTarget `yaml:"targets"`
	Flags   []string      `yaml:"flags"`
	LDFlags string        `yaml:"ldflags"`
	CGO     bool          `yaml:"cgo"`
	Env     map[string]string `yaml:"env"`
}

// BuildTarget defines a single platform target.
//
//	target := config.BuildTarget{OS: "darwin", Arch: "arm64"}
type BuildTarget struct {
	OS   string `yaml:"os"`
	Arch string `yaml:"arch"`
}

// PackageManifest defines the structure of .core/manifest.yaml.
// Used by go-scm and go-build for package identity and signing.
//
//	var pkg config.PackageManifest
//	_ = config.LoadManifest(io.Local, ".core/manifest.yaml", &pkg)
type PackageManifest struct {
	Code         string   `yaml:"code"`
	Name         string   `yaml:"name"`
	Module       string   `yaml:"module"`
	Version      string   `yaml:"version"`
	Description  string   `yaml:"description"`
	Licence      string   `yaml:"licence"`
	Sign         string   `yaml:"sign"`
	SignKey      string   `yaml:"sign_key"`
	Dependencies []string `yaml:"dependencies"`
	Tags         []string `yaml:"tags"`
}

// WorkspaceManifest defines the structure of .core/workspace.yaml.
// Used by core workspace setup to declare project dependencies.
//
//	var ws config.WorkspaceManifest
//	_ = config.LoadManifest(io.Local, ".core/workspace.yaml", &ws)
type WorkspaceManifest struct {
	Version      int      `yaml:"version"`
	Dependencies []string `yaml:"dependencies"`
	Active       string   `yaml:"active"`
	PackagesDir  string   `yaml:"packages_dir"`
	Settings     map[string]any `yaml:"settings"`
}

// LoadManifest reads a YAML manifest file from the given medium and decodes
// it into the destination value. Accepts any of the ViewManifest / BuildManifest /
// PackageManifest / WorkspaceManifest types (or any YAML-tagged struct).
//
//	var build config.BuildManifest
//	err := config.LoadManifest(io.Local, ".core/build.yaml", &build)
func LoadManifest(m coreio.Medium, path string, out any) error {
	content, err := m.Read(path)
	if err != nil {
		return coreerr.E("config.LoadManifest", "failed to read manifest: "+path, err)
	}
	if err := yaml.Unmarshal([]byte(content), out); err != nil {
		return coreerr.E("config.LoadManifest", "failed to parse manifest: "+path, err)
	}
	return nil
}
