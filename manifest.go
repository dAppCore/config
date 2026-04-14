package config

import (
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"gopkg.in/yaml.v3"
)

// Known .core/ file names. Consumers should reference these constants rather
// than hard-coding paths so future renames are a single-site change.
//
//	path := filepath.Join(dir, ".core", config.FileBuild)
const (
	FileConfig    = "config.yaml"    // go-config — identity, preferences, feature flags
	FileBuild     = "build.yaml"     // go-build — targets, ldflags, cgo
	FileRelease   = "release.yaml"   // go-build — archive, checksums, publish
	FileTest      = "test.yaml"      // core dev — test framework override
	FileRun       = "run.yaml"       // core dev — dev services, server, env
	FileView      = "view.yaml"      // go-webview / dAppServer — HLCRF slots, permissions
	FileManifest  = "manifest.yaml"  // go-scm — package identity + signature
	FileWorkspace = "workspace.yaml" // core — project dependencies
	FileRepos     = "repos.yaml"     // go-scm — multi-repo registry
	FileAgent     = "agent.yaml"     // core agent — daemon config (user-level)
	FileZone      = "zone.yaml"      // lethernet — network zone (user-level)

	// Directory is the conventional directory name that holds the .core/ files.
	Directory = ".core"
)

// KnownFiles enumerates the canonical .core/ file names in discovery order.
//
//	for _, name := range config.KnownFiles { /* check existence */ }
var KnownFiles = []string{
	FileConfig,
	FileBuild,
	FileRelease,
	FileTest,
	FileRun,
	FileView,
	FileManifest,
	FileWorkspace,
	FileRepos,
}

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
	Version      int            `yaml:"version"`
	Dependencies []string       `yaml:"dependencies"`
	Active       string         `yaml:"active"`
	PackagesDir  string         `yaml:"packages_dir"`
	Settings     map[string]any `yaml:"settings"`
}

// TestManifest defines the structure of .core/test.yaml.
// Used by core dev to override the auto-detected test framework.
//
//	var test config.TestManifest
//	_ = config.LoadManifest(io.Local, ".core/test.yaml", &test)
type TestManifest struct {
	Version  int               `yaml:"version"`
	Commands []TestCommand     `yaml:"commands"`
	Env      map[string]string `yaml:"env"`
}

// TestCommand is a single named test step (unit, types, lint, ...).
//
//	cmd := config.TestCommand{Name: "unit", Run: "vendor/bin/pest"}
type TestCommand struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}

// RunManifest defines the structure of .core/run.yaml.
// Used by core dev to start the project in its intended dev environment.
//
//	var run config.RunManifest
//	_ = config.LoadManifest(io.Local, ".core/run.yaml", &run)
type RunManifest struct {
	Version  int               `yaml:"version"`
	Services []RunService      `yaml:"services"`
	Dev      RunDev            `yaml:"dev"`
	Env      map[string]string `yaml:"env"`
}

// RunService is a backing service (database, cache, mail) started alongside dev.
//
//	svc := config.RunService{Name: "db", Image: "postgres:16", Port: 5432}
type RunService struct {
	Name  string            `yaml:"name"`
	Image string            `yaml:"image"`
	Port  int               `yaml:"port"`
	Env   map[string]string `yaml:"env"`
}

// RunDev is the primary dev-loop process (serve, watch, reload).
//
//	dev := config.RunDev{Command: "php artisan serve", Port: 8000, Watch: []string{"app/"}}
type RunDev struct {
	Command string   `yaml:"command"`
	Port    int      `yaml:"port"`
	Watch   []string `yaml:"watch"`
}

// ReleaseManifest defines the structure of .core/release.yaml.
// Used by go-build to format archives, attach checksums, and publish to GitHub.
//
//	var rel config.ReleaseManifest
//	_ = config.LoadManifest(io.Local, ".core/release.yaml", &rel)
type ReleaseManifest struct {
	Version   int              `yaml:"version"`
	Archive   ReleaseArchive   `yaml:"archive"`
	Checksums bool             `yaml:"checksums"`
	GitHub    ReleaseGitHub    `yaml:"github"`
	Changelog ReleaseChangelog `yaml:"changelog"`
}

// ReleaseArchive describes the output archive format.
//
//	arc := config.ReleaseArchive{Format: "tar.gz", Include: []string{"LICENSE.txt"}}
type ReleaseArchive struct {
	Format  string   `yaml:"format"`
	Include []string `yaml:"include"`
}

// ReleaseGitHub controls the GitHub Releases publish step.
//
//	gh := config.ReleaseGitHub{Draft: false, Prerelease: false}
type ReleaseGitHub struct {
	Draft      bool `yaml:"draft"`
	Prerelease bool `yaml:"prerelease"`
}

// ReleaseChangelog controls changelog generation from conventional commits.
//
//	log := config.ReleaseChangelog{Include: []string{"feat", "fix"}}
type ReleaseChangelog struct {
	Include []string `yaml:"include"`
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
