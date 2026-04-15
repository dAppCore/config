package config

import (
	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
	"strings"

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
	FileIDE       = "ide.yaml"       // ide — editor integration, LSP, formatters
	FilePHP       = "php.yaml"       // core dev — PHP/Laravel settings
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
	FileIDE,
	FilePHP,
}

// ViewManifest defines the structure of .core/view.yaml.
// Used by go-webview and dAppServer to configure window behaviour, permissions,
// and mounted slots.
//
//	var view config.ViewManifest
//	_ = config.LoadManifest(io.Local, ".core/view.yaml", &view)
type ViewManifest struct {
	Version     string          `yaml:"version"`
	Code        string          `yaml:"code"`
	Name        string          `yaml:"name"`
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
	Version int               `yaml:"version"`
	Project BuildProject      `yaml:"project"`
	Build   BuildSettings     `yaml:"build"`
	Targets []BuildTarget     `yaml:"targets"`
	Signing BuildSigning      `yaml:"sign"`
	SDK     BuildSDK          `yaml:"sdk"`
	Name    string            `yaml:"-"`
	Main    string            `yaml:"-"`
	Binary  string            `yaml:"-"`
	Output  string            `yaml:"-"`
	Flags   []string          `yaml:"-"`
	LDFlags string            `yaml:"-"`
	CGO     bool              `yaml:"-"`
	Env     map[string]string `yaml:"env"`
}

// BuildProject describes the source package being built.
type BuildProject struct {
	Name   string `yaml:"name"`
	Main   string `yaml:"main"`
	Binary string `yaml:"binary"`
	Output string `yaml:"output"`
}

// BuildSettings captures the compiler and linker settings for a build.
type BuildSettings struct {
	Type    string   `yaml:"type"`
	CGO     bool     `yaml:"cgo"`
	Flags   []string `yaml:"flags"`
	LDFlags []string `yaml:"ldflags"`
}

// BuildTarget defines a single platform target.
//
//	target := config.BuildTarget{OS: "darwin", Arch: "arm64"}
type BuildTarget struct {
	OS   string `yaml:"os"`
	Arch string `yaml:"arch"`
}

// UnmarshalYAML accepts either the structured `{os, arch}` form or the RFC
// shorthand `linux/amd64` form.
func (t *BuildTarget) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var raw string
		if err := value.Decode(&raw); err != nil {
			return err
		}
		if raw == "" {
			*t = BuildTarget{}
			return nil
		}
		osPart, archPart, ok := strings.Cut(raw, "/")
		if !ok || osPart == "" || archPart == "" {
			return coreerr.E("config.BuildTarget.UnmarshalYAML", "invalid target shorthand: "+raw, nil)
		}
		*t = BuildTarget{OS: osPart, Arch: archPart}
		return nil
	case yaml.MappingNode:
		type alias BuildTarget
		var raw alias
		if err := value.Decode(&raw); err != nil {
			return err
		}
		*t = BuildTarget(raw)
		return nil
	case yaml.AliasNode:
		return value.Decode(t)
	default:
		*t = BuildTarget{}
		return nil
	}
}

// BuildSigning controls artifact signing for build outputs.
type BuildSigning struct {
	Enabled bool              `yaml:"enabled"`
	GPG     BuildSigningGPG   `yaml:"gpg"`
	MacOS   BuildSigningMacOS `yaml:"macos"`
}

// BuildSigningGPG configures GPG signing.
type BuildSigningGPG struct {
	Key string `yaml:"key"`
}

// BuildSigningMacOS configures macOS signing and notarization.
type BuildSigningMacOS struct {
	Identity string `yaml:"identity"`
	Notarize bool   `yaml:"notarize"`
}

// BuildSDK configures SDK generation from an OpenAPI or similar source.
type BuildSDK struct {
	Spec      string   `yaml:"spec"`
	Languages []string `yaml:"languages"`
	Output    string   `yaml:"output"`
	Diff      bool     `yaml:"diff"`
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

// ReposManifest defines the structure of .core/repos.yaml.
// Used by go-scm and `core dev health` for multi-repo workspace operations.
// Lives at the workspace root (e.g. `~/Code/.core/repos.yaml`) and enumerates
// every repository that belongs to the federated monorepo.
//
//	var repos config.ReposManifest
//	_ = config.LoadManifest(io.Local, "~/Code/.core/repos.yaml", &repos)
type ReposManifest struct {
	Version int         `yaml:"version"`
	Org     string      `yaml:"org"`
	Repos   []ReposRepo `yaml:"repos"`
}

// AgentManifest defines the structure of ~/.core/agent.yaml.
// Used by the agent daemon to configure watch roots, schedules, MCP/API
// listeners, and pool sizing for each model backend.
//
//	var agent config.AgentManifest
//	_ = config.LoadManifest(io.Local, "~/.core/agent.yaml", &agent)
type AgentManifest struct {
	Daemon DaemonConfig         `yaml:"daemon"`
	Agents map[string]AgentPool `yaml:"agents"`
}

// DaemonConfig contains the top-level daemon settings for ~/.core/agent.yaml.
type DaemonConfig struct {
	Enabled  bool             `yaml:"enabled"`
	Watch    []string         `yaml:"watch"`
	Schedule []DaemonSchedule `yaml:"schedule"`
	MCP      DaemonMCP        `yaml:"mcp"`
	API      DaemonAPI        `yaml:"api"`
}

// DaemonSchedule defines a single cron-like daemon task.
type DaemonSchedule struct {
	Cron   string `yaml:"cron"`
	Action string `yaml:"action"`
}

// DaemonMCP configures the daemon's MCP listener.
type DaemonMCP struct {
	Port int `yaml:"port"`
}

// DaemonAPI configures the daemon's API listener.
type DaemonAPI struct {
	Port int    `yaml:"port"`
	Bind string `yaml:"bind"`
}

// AgentPool configures the total worker count for a named agent backend.
type AgentPool struct {
	Total int `yaml:"total"`
}

// ZoneManifest defines the structure of ~/.core/zone.yaml.
// Used by lethernet/network tooling to configure identity, chain mode,
// advertised services, and staking.
//
//	var zone config.ZoneManifest
//	_ = config.LoadManifest(io.Local, "~/.core/zone.yaml", &zone)
type ZoneManifest struct {
	Zone ZoneConfig `yaml:"zone"`
}

// ZoneConfig is the root `zone:` object in ~/.core/zone.yaml.
type ZoneConfig struct {
	Name     string       `yaml:"name"`
	Identity string       `yaml:"identity"`
	Chain    ZoneChain    `yaml:"chain"`
	Network  ZoneNetwork  `yaml:"network"`
	Services ZoneServices `yaml:"services"`
	Staking  ZoneStaking  `yaml:"staking"`
}

// ZoneChain configures blockchain connectivity for the zone.
type ZoneChain struct {
	Mode   string `yaml:"mode"`
	Daemon string `yaml:"daemon"`
}

// ZoneNetwork configures network transport settings for the zone.
type ZoneNetwork struct {
	WireGuard ZoneWireGuard `yaml:"wireguard"`
}

// ZoneWireGuard configures the WireGuard listener for the zone.
type ZoneWireGuard struct {
	Interface string `yaml:"interface"`
	Listen    int    `yaml:"listen"`
}

// ZoneServices enumerates the services this zone offers.
type ZoneServices struct {
	VPN     ZoneServiceVPN     `yaml:"vpn"`
	DNS     ZoneServiceToggle  `yaml:"dns"`
	Compute ZoneServiceCompute `yaml:"compute"`
}

// ZoneServiceToggle is a simple enabled/disabled service switch.
type ZoneServiceToggle struct {
	Enabled bool `yaml:"enabled"`
}

// ZoneServiceVPN configures the VPN service advertisement.
type ZoneServiceVPN struct {
	Enabled  bool    `yaml:"enabled"`
	Price    float64 `yaml:"price"`
	Capacity int     `yaml:"capacity"`
}

// ZoneServiceCompute configures the compute service advertisement.
type ZoneServiceCompute struct {
	Enabled bool     `yaml:"enabled"`
	Models  []string `yaml:"models"`
}

// ZoneStaking configures the zone's staking posture.
type ZoneStaking struct {
	Amount int    `yaml:"amount"`
	Tier   string `yaml:"tier"`
}

type buildManifestYAML struct {
	Version int                  `yaml:"version"`
	Project buildManifestProject `yaml:"project"`
	Build   buildManifestBuild   `yaml:"build"`
	Targets []BuildTarget        `yaml:"targets"`
	Signing buildManifestSigning `yaml:"sign"`
	SDK     buildManifestSDK     `yaml:"sdk"`
	Name    string               `yaml:"name"`
	Main    string               `yaml:"main"`
	Binary  string               `yaml:"binary"`
	Output  string               `yaml:"output"`
	Flags   []string             `yaml:"flags"`
	LDFlags buildManifestLDFlags `yaml:"ldflags"`
	CGO     *bool                `yaml:"cgo"`
	Env     map[string]string    `yaml:"env"`
}

type buildManifestProject struct {
	Name   string `yaml:"name"`
	Main   string `yaml:"main"`
	Binary string `yaml:"binary"`
	Output string `yaml:"output"`
}

type buildManifestBuild struct {
	Type    string               `yaml:"type"`
	CGO     *bool                `yaml:"cgo"`
	Flags   []string             `yaml:"flags"`
	LDFlags buildManifestLDFlags `yaml:"ldflags"`
}

type buildManifestSigning struct {
	Enabled bool                      `yaml:"enabled"`
	GPG     buildManifestSigningGPG   `yaml:"gpg"`
	MacOS   buildManifestSigningMacOS `yaml:"macos"`
}

type buildManifestSigningGPG struct {
	Key string `yaml:"key"`
}

type buildManifestSigningMacOS struct {
	Identity string `yaml:"identity"`
	Notarize bool   `yaml:"notarize"`
}

type buildManifestSDK struct {
	Spec      string   `yaml:"spec"`
	Languages []string `yaml:"languages"`
	Output    string   `yaml:"output"`
	Diff      bool     `yaml:"diff"`
}

type buildManifestLDFlags []string

func (l *buildManifestLDFlags) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		var single string
		if err := value.Decode(&single); err != nil {
			return err
		}
		if single == "" {
			*l = nil
			return nil
		}
		*l = []string{single}
		return nil
	case yaml.SequenceNode:
		var values []string
		if err := value.Decode(&values); err != nil {
			return err
		}
		*l = append([]string(nil), values...)
		return nil
	case yaml.MappingNode, yaml.AliasNode:
		var values []string
		if err := value.Decode(&values); err != nil {
			return err
		}
		*l = append([]string(nil), values...)
		return nil
	default:
		*l = nil
		return nil
	}
}

func (l buildManifestLDFlags) String() string {
	return strings.Join(l, " ")
}

// UnmarshalYAML accepts both the legacy flat build schema and the nested
// RFC shape with project/build sections.
func (m *BuildManifest) UnmarshalYAML(value *yaml.Node) error {
	var raw buildManifestYAML
	if err := value.Decode(&raw); err != nil {
		return err
	}

	m.Version = raw.Version
	m.Project = BuildProject{
		Name:   firstNonEmpty(raw.Project.Name, raw.Name),
		Main:   firstNonEmpty(raw.Project.Main, raw.Main),
		Binary: firstNonEmpty(raw.Project.Binary, raw.Binary),
		Output: firstNonEmpty(raw.Project.Output, raw.Output),
	}
	m.Build = BuildSettings{
		Type:    raw.Build.Type,
		CGO:     firstBool(raw.Build.CGO, raw.CGO),
		Flags:   firstStrings(raw.Build.Flags, raw.Flags),
		LDFlags: firstLDFlags(raw.Build.LDFlags, raw.LDFlags),
	}
	m.Targets = append([]BuildTarget(nil), raw.Targets...)
	m.Signing = BuildSigning{
		Enabled: raw.Signing.Enabled,
		GPG: BuildSigningGPG{
			Key: raw.Signing.GPG.Key,
		},
		MacOS: BuildSigningMacOS{
			Identity: raw.Signing.MacOS.Identity,
			Notarize: raw.Signing.MacOS.Notarize,
		},
	}
	m.SDK = BuildSDK{
		Spec:      raw.SDK.Spec,
		Languages: append([]string(nil), raw.SDK.Languages...),
		Output:    raw.SDK.Output,
		Diff:      raw.SDK.Diff,
	}
	m.Name = m.Project.Name
	m.Main = m.Project.Main
	m.Binary = m.Project.Binary
	m.Output = m.Project.Output
	m.Flags = append([]string(nil), m.Build.Flags...)
	m.LDFlags = strings.Join(m.Build.LDFlags, " ")
	m.CGO = m.Build.CGO
	m.Env = raw.Env
	return nil
}

// ReposRepo is a single repository entry in repos.yaml.
//
//	repo := config.ReposRepo{Path: "core/go", Remote: "ssh://…/go.git", Branch: "dev"}
type ReposRepo struct {
	Path        string   `yaml:"path"`
	Remote      string   `yaml:"remote"`
	Branch      string   `yaml:"branch"`
	Type        string   `yaml:"type"`
	Description string   `yaml:"description"`
	Depends     []string `yaml:"depends"`
}

// LoadManifest reads a YAML manifest file from the given medium and decodes
// it into the destination value. Accepts any of the ViewManifest / BuildManifest /
// PackageManifest / WorkspaceManifest / ReposManifest types (or any YAML-tagged
// struct).
//
//	var build config.BuildManifest
//	err := config.LoadManifest(io.Local, ".core/build.yaml", &build)
func LoadManifest(m coreio.Medium, path string, out any) error {
	content, err := m.Read(path)
	if err != nil {
		return coreerr.E("config.LoadManifest", "failed to read manifest: "+path, err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
		return coreerr.E("config.LoadManifest", "failed to parse manifest: "+path, err)
	}
	if err := validateSchema(path, raw); err != nil {
		return err
	}
	if err := yaml.Unmarshal([]byte(content), out); err != nil {
		return coreerr.E("config.LoadManifest", "failed to decode manifest: "+path, err)
	}
	if err := validateManifest(path, out, raw); err != nil {
		return err
	}
	return nil
}

func validateManifest(path string, out any, raw map[string]any) error {
	switch core.PathBase(path) {
	case FileView:
		view, ok := out.(*ViewManifest)
		if !ok {
			return nil
		}
		if hasEmptyStringField(raw, "sign") && view.Sign == "" {
			return coreerr.E("config.LoadManifest", "unsigned view manifest rejected: "+path, nil)
		}
	case FileManifest:
		pkg, ok := out.(*PackageManifest)
		if !ok {
			return nil
		}
		if hasEmptyStringField(raw, "sign") && pkg.Sign == "" {
			return coreerr.E("config.LoadManifest", "unsigned package manifest rejected: "+path, nil)
		}
		if hasEmptyStringField(raw, "sign_key") && pkg.SignKey == "" {
			return coreerr.E("config.LoadManifest", "missing package sign_key: "+path, nil)
		}
	}
	return nil
}

func hasEmptyStringField(raw map[string]any, key string) bool {
	value, ok := raw[key]
	if !ok {
		return false
	}
	s, ok := value.(string)
	return ok && strings.TrimSpace(s) == ""
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}

func firstBool(values ...*bool) bool {
	for _, value := range values {
		if value != nil {
			return *value
		}
	}
	return false
}

func firstStrings(values ...[]string) []string {
	for _, value := range values {
		if len(value) > 0 {
			return append([]string(nil), value...)
		}
	}
	return nil
}

func firstLDFlags(values ...buildManifestLDFlags) buildManifestLDFlags {
	for _, value := range values {
		if len(value) > 0 {
			return append(buildManifestLDFlags(nil), value...)
		}
	}
	return nil
}
