package config

import (
	"crypto/ed25519"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
	"gopkg.in/yaml.v3"
)

// Known .core/ file names. Consumers should reference these constants rather
// than hard-coding paths so future renames are a single-site change.
//
//	path := filepath.Join(dir, ".core", config.FileBuild)
const (
	FileConfig         = "config.yaml"    // go-config — identity, preferences, feature flags
	FileBuild          = "build.yaml"     // go-build — targets, ldflags, cgo
	FileRelease        = "release.yaml"   // go-build — archive, checksums, publish
	FileTest           = "test.yaml"      // core dev — test framework override
	FileRun            = "run.yaml"       // core dev — dev services, server, env
	FileView           = "view.yaml"      // go-webview / dAppServer — HLCRF slots, permissions
	FileManifest       = "manifest.yaml"  // go-scm — package identity + signature
	FileWorkspace      = "workspace.yaml" // core — project dependencies
	FileRepos          = "repos.yaml"     // go-scm — multi-repo registry
	FileIDE            = "ide.yaml"       // ide — editor integration, LSP, formatters
	FilePHP            = "php.yaml"       // core dev — PHP/Laravel settings
	FileAgent          = "agent.yaml"     // core agent — daemon config (user-level)
	FileZone           = "zone.yaml"      // lethernet — network zone (user-level)
	FileImagesManifest = "manifest.json"  // core dev — LinuxKit image registry
	FileLinuxKit       = "core-dev.yml"   // core dev — LinuxKit base image config

	// Directory is the conventional directory name that holds the .core/ files.
	Directory = ".core"

	// Directory names that live under ~/.core/ for user-level registries.
	DirectoryImages     = "images"
	DirectorySecrets    = "secrets"
	DirectoryDaemons    = "daemons"
	DirectoryWorkspaces = "workspaces"

	// WorkspaceDirectory is the sandbox root inside a project-local .core/.
	WorkspaceDirectory = "workspace"

	// WorkspaceSourceDirectory is the checked-out repository source inside a
	// sandboxed workspace.
	WorkspaceSourceDirectory = "src"

	// WorkspaceMetaDirectory stores agent logs, status files, and other
	// workspace-local bookkeeping.
	WorkspaceMetaDirectory = ".meta"

	// WorkspaceInstructionsFile is the agent instruction file stored at the
	// root of a sandboxed workspace.
	WorkspaceInstructionsFile = "CODEX.md"

	// LinuxKitDirectory is the conventional directory for LinuxKit templates
	// under either a project-local or user-global .core/ tree.
	LinuxKitDirectory = "linuxkit"
)

const (
	callerLoadManifest                  = "config.LoadManifest"
	callerTrustedManifestPublicKeys     = "config.trustedManifestPublicKeys"
	callerValidateViewManifestSignature = "config.ValidateViewManifestSignature"
	callerVerifyViewManifestSignature   = "config.VerifyViewManifestSignature"
	callerSignViewManifest              = "config.SignViewManifest"
	callerSignPackageManifest           = "config.SignPackageManifest"
	callerVerifyPackageManifest         = "config.VerifyPackageManifest"
	callerParseManifestPublicKey        = "config.parseManifestPublicKey"

	errCanonicalMarshalFailed = "canonical marshal failed"
	errDecodeTrustedKeyFailed = "decode trusted key failed"
	manifestTargetOSKey       = "o" + "s"
)

var manifestHomeDir = func() string {
	return core.Env("DIR_HOME")
}

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
	FileAgent,
	FileZone,
}

// ViewManifest defines the structure of .core/view.yaml.
// Used by go-webview and dAppServer to configure window behaviour, permissions,
// and mounted slots.
//
//	var view config.ViewManifest
//	_ = config.LoadManifest(io.Local, ".core/view.yaml", &view)
type ViewManifest struct {
	Version     ViewVersion     `yaml:"version"`
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

// ViewVersion accepts either the folder-spec integer form (`version: 1`) or
// the RFC example's semantic string form (`version: 0.1.0`).
type ViewVersion string

// UnmarshalYAML keeps view.yaml backward-compatible across the RFC's mixed
// version examples while preserving the public string-shaped API.
func (v *ViewVersion) UnmarshalYAML(node *yaml.Node) core.Result {
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		return v.UnmarshalYAML(node.Alias)
	}
	switch node.Kind {
	case yaml.ScalarNode:
		var asString string
		if err := node.Decode(&asString); err == nil {
			*v = ViewVersion(asString)
			return core.Ok(nil)
		}
		var asInt int
		if err := node.Decode(&asInt); err == nil {
			*v = ViewVersion(core.Sprintf("%d", asInt))
			return core.Ok(nil)
		}
	}
	return core.Fail(coreerr.E("config.ViewVersion.UnmarshalYAML", "invalid view manifest version", nil))
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
	OS   string
	Arch string `yaml:"arch"`
}

// UnmarshalYAML accepts either the structured `{os, arch}` form or the RFC
// shorthand `linux/amd64` form.
func (t *BuildTarget) UnmarshalYAML(value *yaml.Node) core.Result {
	if value.Kind == yaml.AliasNode && value.Alias != nil {
		return t.UnmarshalYAML(value.Alias)
	}
	switch value.Kind {
	case yaml.ScalarNode:
		var raw string
		if err := value.Decode(&raw); err != nil {
			return core.Fail(err)
		}
		if raw == "" {
			*t = BuildTarget{}
			return core.Ok(nil)
		}
		parsed := buildTargetFromString(raw)
		if !parsed.OK {
			return parsed
		}
		*t = parsed.Value.(BuildTarget)
		return core.Ok(nil)
	case yaml.MappingNode:
		type alias BuildTarget
		var raw alias
		if err := value.Decode(&raw); err != nil {
			return core.Fail(err)
		}
		*t = BuildTarget(raw)
		return core.Ok(nil)
	default:
		*t = BuildTarget{}
		return core.Ok(nil)
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

// IDEManifest defines the structure of .core/ide.yaml.
// Used by editor and LSP integrations to discover workspace-local IDE hints.
//
//	var ide config.IDEManifest
//	_ = config.LoadManifest(io.Local, ".core/ide.yaml", &ide)
type IDEManifest struct {
	Version  int            `yaml:"version"`
	Editor   string         `yaml:"editor"`
	LSP      map[string]any `yaml:"lsp"`
	Format   map[string]any `yaml:"format"`
	Tasks    map[string]any `yaml:"tasks"`
	Settings map[string]any `yaml:"settings"`
}

// PHPManifest defines the structure of .core/php.yaml.
// Used by core dev / core php commands to configure the local PHP runtime,
// test runner, lint tooling, and optional deploy integration.
//
//	var php config.PHPManifest
//	_ = config.LoadManifest(io.Local, ".core/php.yaml", &php)
type PHPManifest struct {
	Version int       `yaml:"version"`
	Server  PHPServer `yaml:"server"`
	Test    PHPTest   `yaml:"test"`
	Lint    PHPLint   `yaml:"lint"`
	Deploy  PHPDeploy `yaml:"deploy"`
}

// PHPServer configures the dev server used by `core php serve`.
type PHPServer struct {
	Type    string `yaml:"type"`
	Port    int    `yaml:"port"`
	Workers int    `yaml:"workers"`
}

// PHPTest configures the PHP test runner used by `core php test`.
type PHPTest struct {
	Framework string `yaml:"framework"`
	Parallel  bool   `yaml:"parallel"`
}

// PHPLint configures the lint tool used by `core php lint`.
type PHPLint struct {
	Tool   string `yaml:"tool"`
	Config string `yaml:"config"`
}

// PHPDeploy configures optional PHP deploy settings for higher-level tooling.
type PHPDeploy struct {
	Type         string            `yaml:"type"`
	Environment  string            `yaml:"environment"`
	Command      string            `yaml:"command"`
	Inventory    string            `yaml:"inventory"`
	Environments map[string]string `yaml:"environments"`
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
	Targets []any                `yaml:"targets"`
	Signing buildManifestSigning `yaml:"sign"`
	SDK     buildManifestSDK     `yaml:"sdk"`
	Name    string               `yaml:"name"`
	Main    string               `yaml:"main"`
	Binary  string               `yaml:"binary"`
	Output  string               `yaml:"output"`
	Flags   []string             `yaml:"flags"`
	LDFlags any                  `yaml:"ldflags"`
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
	Type    string   `yaml:"type"`
	CGO     *bool    `yaml:"cgo"`
	Flags   []string `yaml:"flags"`
	LDFlags any      `yaml:"ldflags"`
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

type buildmanifestldflags []string

func (l *buildmanifestldflags) UnmarshalYAML(value *yaml.Node) core.Result {
	if value.Kind == yaml.AliasNode && value.Alias != nil {
		return l.UnmarshalYAML(value.Alias)
	}
	switch value.Kind {
	case yaml.ScalarNode:
		var single string
		if err := value.Decode(&single); err != nil {
			return core.Fail(err)
		}
		if single == "" {
			*l = nil
			return core.Ok(nil)
		}
		*l = []string{single}
		return core.Ok(nil)
	case yaml.SequenceNode:
		var values []string
		if err := value.Decode(&values); err != nil {
			return core.Fail(err)
		}
		*l = append([]string(nil), values...)
		return core.Ok(nil)
	case yaml.MappingNode:
		return core.Fail(coreerr.E("config.buildmanifestldflags.UnmarshalYAML", "unsupported ldflags mapping", nil))
	default:
		*l = nil
		return core.Ok(nil)
	}
}

func (l buildmanifestldflags) String() string {
	return core.Join(" ", l...)
}

// UnmarshalYAML accepts both the legacy flat build schema and the nested
// RFC shape with project/build sections.
func (m *BuildManifest) UnmarshalYAML(value *yaml.Node) core.Result {
	if value.Kind == yaml.AliasNode && value.Alias != nil {
		return m.UnmarshalYAML(value.Alias)
	}
	var raw buildManifestYAML
	if err := value.Decode(&raw); err != nil {
		return core.Fail(err)
	}

	targetsResult := buildTargetsFromYAML(raw.Targets)
	if !targetsResult.OK {
		return targetsResult
	}
	buildLDFlagsResult := buildLDFlagsFromYAML(raw.Build.LDFlags)
	if !buildLDFlagsResult.OK {
		return buildLDFlagsResult
	}
	legacyLDFlagsResult := buildLDFlagsFromYAML(raw.LDFlags)
	if !legacyLDFlagsResult.OK {
		return legacyLDFlagsResult
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
		LDFlags: firstLDFlags(buildLDFlagsResult.Value.(buildmanifestldflags), legacyLDFlagsResult.Value.(buildmanifestldflags)),
	}
	m.Targets = targetsResult.Value.([]BuildTarget)
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
	m.LDFlags = core.Join(" ", m.Build.LDFlags...)
	m.CGO = m.Build.CGO
	m.Env = raw.Env
	return core.Ok(nil)
}

func buildTargetFromString(raw string) core.Result {
	parts := core.SplitN(raw, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return core.Fail(coreerr.E("config.BuildTarget.UnmarshalYAML", "invalid target shorthand: "+raw, nil))
	}
	return core.Ok(BuildTarget{OS: parts[0], Arch: parts[1]})
}

func buildTargetsFromYAML(values []any) core.Result {
	targets := make([]BuildTarget, 0, len(values))
	for _, value := range values {
		targetResult := buildTargetFromYAML(value)
		if !targetResult.OK {
			return targetResult
		}
		targets = append(targets, targetResult.Value.(BuildTarget))
	}
	return core.Ok(targets)
}

func buildTargetFromYAML(value any) core.Result {
	switch typed := value.(type) {
	case string:
		if typed == "" {
			return core.Ok(BuildTarget{})
		}
		return buildTargetFromString(typed)
	case map[string]any:
		return core.Ok(BuildTarget{
			OS:   stringFromYAMLMap(typed, manifestTargetOSKey),
			Arch: stringFromYAMLMap(typed, "arch"),
		})
	case map[any]any:
		return core.Ok(BuildTarget{
			OS:   stringFromAnyYAMLMap(typed, manifestTargetOSKey),
			Arch: stringFromAnyYAMLMap(typed, "arch"),
		})
	case nil:
		return core.Ok(BuildTarget{})
	default:
		return core.Fail(coreerr.E("config.BuildTarget.UnmarshalYAML", "invalid target entry", nil))
	}
}

func stringFromYAMLMap(values map[string]any, key string) string {
	if value, ok := values[key].(string); ok {
		return value
	}
	return ""
}

func stringFromAnyYAMLMap(values map[any]any, key string) string {
	if value, ok := values[key]; ok {
		if s, ok := value.(string); ok {
			return s
		}
	}
	return ""
}

func buildLDFlagsFromYAML(value any) core.Result {
	switch typed := value.(type) {
	case nil:
		return core.Ok(buildmanifestldflags(nil))
	case string:
		if typed == "" {
			return core.Ok(buildmanifestldflags(nil))
		}
		return core.Ok(buildmanifestldflags{typed})
	case []string:
		return core.Ok(buildmanifestldflags(append([]string(nil), typed...)))
	case []any:
		out := make(buildmanifestldflags, 0, len(typed))
		for _, value := range typed {
			s, ok := value.(string)
			if !ok {
				return core.Fail(coreerr.E("config.buildmanifestldflags.UnmarshalYAML", "invalid ldflags sequence", nil))
			}
			out = append(out, s)
		}
		return core.Ok(out)
	case map[string]any, map[any]any:
		return core.Fail(coreerr.E("config.buildmanifestldflags.UnmarshalYAML", "unsupported ldflags mapping", nil))
	default:
		return core.Fail(coreerr.E("config.buildmanifestldflags.UnmarshalYAML", "invalid ldflags value", nil))
	}
}

// ReposRepo is a single repository entry in repos.yaml.
//
//	repo := config.ReposRepo{Path: "core/go", Remote: "ssh://…/go.git", Branch: "dev"}
type ReposRepo struct {
	Path        string
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
func LoadManifest(m coreio.Medium, path string, out any) core.Result {
	content, err := m.Read(path)
	if err != nil {
		return core.Fail(coreerr.E(callerLoadManifest, "failed to read manifest: "+path, err))
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
		return core.Fail(coreerr.E(callerLoadManifest, "failed to parse manifest: "+path, err))
	}
	if r := validateSchema(path, raw); !r.OK {
		return r
	}
	if r := decodeManifestYAML(content, out); !r.OK {
		return core.Fail(coreerr.E(callerLoadManifest, "failed to decode manifest: "+path, resultCause(r).(error)))
	}
	if r := validateManifest(path, out, raw); !r.OK {
		return r
	}
	return core.Ok(nil)
}

func decodeManifestYAML(content string, out any) core.Result {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(content), &node); err != nil {
		return core.Fail(err)
	}
	root := manifestYAMLRoot(&node)
	switch target := out.(type) {
	case *BuildManifest:
		return target.UnmarshalYAML(root)
	case *ViewManifest:
		return decodeViewManifestYAML(root, target)
	default:
		if err := yaml.Unmarshal([]byte(content), out); err != nil {
			return core.Fail(err)
		}
		return core.Ok(nil)
	}
}

func manifestYAMLRoot(node *yaml.Node) *yaml.Node {
	if node == nil {
		return &yaml.Node{}
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return manifestYAMLRoot(node.Content[0])
	}
	if node.Kind == yaml.AliasNode && node.Alias != nil {
		return manifestYAMLRoot(node.Alias)
	}
	return node
}

type viewManifestYAML struct {
	Version     any             `yaml:"version"`
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

func decodeViewManifestYAML(value *yaml.Node, view *ViewManifest) core.Result {
	var raw viewManifestYAML
	if err := value.Decode(&raw); err != nil {
		return core.Fail(err)
	}
	versionResult := viewVersionFromYAML(raw.Version)
	if !versionResult.OK {
		return versionResult
	}
	view.Version = versionResult.Value.(ViewVersion)
	view.Code = raw.Code
	view.Name = raw.Name
	view.Sign = raw.Sign
	view.Title = raw.Title
	view.Width = raw.Width
	view.Height = raw.Height
	view.Resizable = raw.Resizable
	view.Layout = raw.Layout
	view.Slots = raw.Slots
	view.Modules = append([]string(nil), raw.Modules...)
	view.Permissions = raw.Permissions
	view.Config = raw.Config
	return core.Ok(nil)
}

func viewVersionFromYAML(value any) core.Result {
	switch typed := value.(type) {
	case nil:
		return core.Ok(ViewVersion(""))
	case string:
		return core.Ok(ViewVersion(typed))
	case int:
		return core.Ok(ViewVersion(core.Sprintf("%d", typed)))
	case int64:
		return core.Ok(ViewVersion(core.Sprintf("%d", typed)))
	case float64:
		return core.Ok(ViewVersion(core.Sprintf("%v", typed)))
	default:
		return core.Fail(coreerr.E("config.ViewVersion.UnmarshalYAML", "invalid view manifest version", nil))
	}
}

func validateManifest(path string, out any, raw map[string]any) core.Result {
	switch core.PathBase(path) {
	case FileView:
		view, ok := out.(*ViewManifest)
		if !ok {
			return core.Ok(nil)
		}
		if r := validateLoadedViewManifest(path, view, raw); !r.OK {
			return r
		}
	case FileManifest:
		pkg, ok := out.(*PackageManifest)
		if !ok {
			return core.Ok(nil)
		}
		if r := verifyLoadedPackageManifest(path, pkg, raw); !r.OK {
			return r
		}
	}
	return core.Ok(nil)
}

func validateLoadedViewManifest(path string, view *ViewManifest, raw map[string]any) core.Result {
	if missingOrEmptyStringField(raw, "sign", view.Sign) {
		return core.Fail(coreerr.E(callerLoadManifest, "unsigned view manifest rejected: "+path, nil))
	}
	if r := ValidateViewManifestSignature(view); !r.OK {
		msg := r.Error()
		switch {
		case core.Contains(msg, "not ed25519-sized"):
			return core.Fail(coreerr.E(callerLoadManifest, "view manifest signature is not ed25519-sized: "+path, nil))
		case core.Contains(msg, "unsigned"):
			return core.Fail(coreerr.E(callerLoadManifest, "unsigned view manifest rejected: "+path, nil))
		default:
			return core.Fail(coreerr.E(callerLoadManifest, "invalid view manifest signature: "+path, resultCause(r).(error)))
		}
	}
	return core.Ok(nil)
}

func verifyLoadedPackageManifest(path string, pkg *PackageManifest, raw map[string]any) core.Result {
	if missingOrEmptyStringField(raw, "sign", pkg.Sign) {
		return core.Fail(coreerr.E(callerLoadManifest, "unsigned package manifest rejected: "+path, nil))
	}
	if missingOrEmptyStringField(raw, "sign_key", pkg.SignKey) {
		return core.Fail(coreerr.E(callerLoadManifest, "missing package sign_key: "+path, nil))
	}
	if r := VerifyPackageManifest(pkg); !r.OK {
		msg := r.Error()
		switch {
		case core.Contains(msg, "missing package sign_key"):
			return core.Fail(coreerr.E(callerLoadManifest, "missing package sign_key: "+path, nil))
		case core.Contains(msg, "not an ed25519 public key"):
			return core.Fail(coreerr.E(callerLoadManifest, "package sign_key is not an ed25519 public key: "+path, nil))
		case core.Contains(msg, "not trusted"):
			return core.Fail(coreerr.E(callerLoadManifest, "package sign_key is not trusted: "+path, nil))
		case core.Contains(msg, "not ed25519-sized"):
			return core.Fail(coreerr.E(callerLoadManifest, "package manifest signature is not ed25519-sized: "+path, nil))
		case core.Contains(msg, "signature mismatch"):
			return core.Fail(coreerr.E(callerLoadManifest, "package manifest signature mismatch: "+path, nil))
		case core.Contains(msg, errCanonicalMarshalFailed):
			return core.Fail(coreerr.E(callerLoadManifest, errCanonicalMarshalFailed+": "+path, resultCause(r).(error)))
		case core.Contains(msg, "decode package sign_key failed"):
			return core.Fail(coreerr.E(callerLoadManifest, "decode package sign_key failed: "+path, resultCause(r).(error)))
		default:
			return core.Fail(coreerr.E(callerLoadManifest, "invalid package manifest signature: "+path, resultCause(r).(error)))
		}
	}
	return core.Ok(nil)
}

func manifestKeyTrusted(candidate ed25519.PublicKey, trusted []ed25519.PublicKey) bool {
	for _, key := range trusted {
		if len(key) != ed25519.PublicKeySize {
			continue
		}
		if core.Lower(core.HexEncode(candidate)) == core.Lower(core.HexEncode(key)) {
			return true
		}
	}
	return false
}

func trustedManifestTrustedEnvKeys() core.Result {
	fromEnv := core.Trim(core.Env("CORE_MANIFEST_TRUST_KEYS"))
	if fromEnv == "" {
		return core.Ok([]ed25519.PublicKey(nil))
	}
	return parseTrustedManifestKeyList(fromEnv)
}

func decodeManifestSignature(value string) core.Result {
	return core.Base64Decode(core.Trim(value))
}

// CanonicalViewManifestBytes returns the RFC canonical view manifest body with
// the sign field cleared so callers can sign or verify it consistently.
//
//	body, _ := config.CanonicalViewManifestBytes(&view)
func CanonicalViewManifestBytes(view *ViewManifest) core.Result {
	return viewManifestBytes(view)
}

// ValidateViewManifestSignature checks only that view.yaml carries a base64
// ed25519-sized signature. Trust-root verification belongs to the caller.
//
//	if err := config.ValidateViewManifestSignature(&view); err != nil { ... }
func ValidateViewManifestSignature(view *ViewManifest) core.Result {
	if view == nil || core.Trim(view.Sign) == "" {
		return core.Fail(coreerr.E(callerValidateViewManifestSignature, "unsigned view manifest rejected", nil))
	}
	sigResult := decodeManifestSignature(view.Sign)
	if !sigResult.OK {
		return core.Fail(coreerr.E(callerValidateViewManifestSignature, "invalid view manifest signature", resultCause(sigResult).(error)))
	}
	sig := sigResult.Value.([]byte)
	if len(sig) != ed25519.SignatureSize {
		return core.Fail(coreerr.E(callerValidateViewManifestSignature, "view manifest signature is not ed25519-sized", nil))
	}
	return core.Ok(nil)
}

// VerifyViewManifestSignature verifies a signed view manifest against the
// caller-supplied ed25519 public key.
//
//	if err := config.VerifyViewManifestSignature(&view, pub); err != nil { ... }
func VerifyViewManifestSignature(view *ViewManifest, publicKey ed25519.PublicKey) core.Result {
	if r := ValidateViewManifestSignature(view); !r.OK {
		return r
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return core.Fail(coreerr.E(callerVerifyViewManifestSignature, "view manifest public key is not an ed25519 public key", nil))
	}
	bodyResult := CanonicalViewManifestBytes(view)
	if !bodyResult.OK {
		return core.Fail(coreerr.E(callerVerifyViewManifestSignature, errCanonicalMarshalFailed, resultCause(bodyResult).(error)))
	}
	body := bodyResult.Value.([]byte)
	sigResult := decodeManifestSignature(view.Sign)
	if !sigResult.OK {
		return core.Fail(coreerr.E(callerVerifyViewManifestSignature, "invalid view manifest signature", resultCause(sigResult).(error)))
	}
	sig := sigResult.Value.([]byte)
	if !ed25519.Verify(publicKey, body, sig) {
		return core.Fail(coreerr.E(callerVerifyViewManifestSignature, "view manifest signature mismatch", nil))
	}
	return core.Ok(nil)
}

// SignViewManifest signs view.yaml in place using the RFC canonical body and
// stores the resulting base64 signature in Sign.
//
//	_ = config.SignViewManifest(&view, priv)
func SignViewManifest(view *ViewManifest, privateKey ed25519.PrivateKey) core.Result {
	if view == nil {
		return core.Fail(coreerr.E(callerSignViewManifest, "nil view manifest", nil))
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return core.Fail(coreerr.E(callerSignViewManifest, "view manifest private key is not an ed25519 private key", nil))
	}
	bodyResult := CanonicalViewManifestBytes(view)
	if !bodyResult.OK {
		return core.Fail(coreerr.E(callerSignViewManifest, errCanonicalMarshalFailed, resultCause(bodyResult).(error)))
	}
	body := bodyResult.Value.([]byte)
	view.Sign = core.Base64Encode(ed25519.Sign(privateKey, body))
	return core.Ok(nil)
}

func viewManifestBytes(view *ViewManifest) core.Result {
	if view == nil {
		return core.ResultOf(yaml.Marshal(nil))
	}
	tmp := *view
	tmp.Sign = ""
	return core.ResultOf(yaml.Marshal(&tmp))
}

// CanonicalPackageManifestBytes returns the RFC canonical package manifest body
// with the sign field cleared so callers can sign or verify it consistently.
//
//	body, _ := config.CanonicalPackageManifestBytes(&pkg)
func CanonicalPackageManifestBytes(pkg *PackageManifest) core.Result {
	return packageManifestBytes(pkg)
}

// SignPackageManifest signs manifest.yaml in place and ensures SignKey matches
// the supplied ed25519 private key's public half.
//
//	_ = config.SignPackageManifest(&pkg, priv)
func SignPackageManifest(pkg *PackageManifest, privateKey ed25519.PrivateKey) core.Result {
	if pkg == nil {
		return core.Fail(coreerr.E(callerSignPackageManifest, "nil package manifest", nil))
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return core.Fail(coreerr.E(callerSignPackageManifest, "package manifest private key is not an ed25519 private key", nil))
	}
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok || len(publicKey) != ed25519.PublicKeySize {
		return core.Fail(coreerr.E(callerSignPackageManifest, "derive package manifest public key failed", nil))
	}
	pkg.SignKey = core.HexEncode(publicKey)
	bodyResult := CanonicalPackageManifestBytes(pkg)
	if !bodyResult.OK {
		return core.Fail(coreerr.E(callerSignPackageManifest, errCanonicalMarshalFailed, resultCause(bodyResult).(error)))
	}
	body := bodyResult.Value.([]byte)
	pkg.Sign = core.Base64Encode(ed25519.Sign(privateKey, body))
	return core.Ok(nil)
}

func packageManifestBytes(pkg *PackageManifest) core.Result {
	if pkg == nil {
		return core.ResultOf(yaml.Marshal(nil))
	}
	tmp := *pkg
	tmp.Sign = ""
	return core.ResultOf(yaml.Marshal(&tmp))
}

// VerifyPackageManifest verifies manifest.yaml against its embedded sign_key
// and the optional trust roots from CORE_MANIFEST_TRUST_KEYS / ~/.core/keys.
//
//	if err := config.VerifyPackageManifest(&pkg); err != nil { ... }
func VerifyPackageManifest(pkg *PackageManifest) core.Result {
	if pkg == nil || core.Trim(pkg.Sign) == "" {
		return core.Fail(coreerr.E(callerVerifyPackageManifest, "unsigned package manifest rejected", nil))
	}
	if core.Trim(pkg.SignKey) == "" {
		return core.Fail(coreerr.E(callerVerifyPackageManifest, "missing package sign_key", nil))
	}

	pubResult := core.HexDecode(core.Trim(pkg.SignKey))
	if !pubResult.OK {
		return core.Fail(coreerr.E(callerVerifyPackageManifest, "decode package sign_key failed", resultCause(pubResult).(error)))
	}
	pub := pubResult.Value.([]byte)
	if len(pub) != ed25519.PublicKeySize {
		return core.Fail(coreerr.E(callerVerifyPackageManifest, "package sign_key is not an ed25519 public key", nil))
	}

	trustedKeysResult := trustedManifestVerificationKeys()
	if !trustedKeysResult.OK {
		return core.Fail(coreerr.E(callerVerifyPackageManifest, "load trusted manifest public keys failed", resultCause(trustedKeysResult).(error)))
	}
	trustedKeys := trustedKeysResult.Value.([]ed25519.PublicKey)
	if len(trustedKeys) > 0 && !manifestKeyTrusted(ed25519.PublicKey(pub), trustedKeys) {
		return core.Fail(coreerr.E(callerVerifyPackageManifest, "package sign_key is not trusted", nil))
	}

	sigResult := decodeManifestSignature(pkg.Sign)
	if !sigResult.OK {
		return core.Fail(coreerr.E(callerVerifyPackageManifest, "invalid package manifest signature", resultCause(sigResult).(error)))
	}
	sig := sigResult.Value.([]byte)
	if len(sig) != ed25519.SignatureSize {
		return core.Fail(coreerr.E(callerVerifyPackageManifest, "package manifest signature is not ed25519-sized", nil))
	}

	bodyResult := CanonicalPackageManifestBytes(pkg)
	if !bodyResult.OK {
		return core.Fail(coreerr.E(callerVerifyPackageManifest, errCanonicalMarshalFailed, resultCause(bodyResult).(error)))
	}
	body := bodyResult.Value.([]byte)
	if !ed25519.Verify(ed25519.PublicKey(pub), body, sig) {
		return core.Fail(coreerr.E(callerVerifyPackageManifest, "package manifest signature mismatch", nil))
	}
	return core.Ok(nil)
}

// TrustedManifestPublicKeys returns the deduplicated trust roots discovered
// from CORE_MANIFEST_TRUST_KEYS or ~/.core/keys/*.pub.
//
//	keys, _ := config.TrustedManifestPublicKeys()
func TrustedManifestPublicKeys() core.Result {
	return trustedManifestPublicKeys()
}

func trustedManifestPublicKeys() core.Result {
	if fromEnv := core.Trim(core.Env("CORE_MANIFEST_TRUST_KEYS")); fromEnv != "" {
		return parseTrustedManifestKeyList(fromEnv)
	}

	return trustedManifestPublicKeysFromDisk(manifestHomeDir())
}

func parseTrustedManifestKeyList(raw string) core.Result {
	var keys []ed25519.PublicKey
	for _, item := range splitManifestTrustedKeys(raw) {
		pubResult := parseManifestPublicKey(item)
		if !pubResult.OK {
			return core.Fail(coreerr.E(callerTrustedManifestPublicKeys, errDecodeTrustedKeyFailed, resultCause(pubResult).(error)))
		}
		keys = append(keys, pubResult.Value.(ed25519.PublicKey))
	}
	return core.Ok(dedupeManifestKeys(keys))
}

func trustedManifestPublicKeysFromDisk(home string) core.Result {
	if home == "" {
		return core.Ok([]ed25519.PublicKey(nil))
	}

	coreDir := core.PathJoin(home, ".core")
	keyDir := core.PathJoin(coreDir, "keys")
	if isSymlinkedCoreDir(coreio.Local, coreDir) {
		return core.Fail(coreerr.E(callerTrustedManifestPublicKeys, "symlinked .core directory rejected: "+coreDir, nil))
	}
	if isSymlinkedLocalPath(keyDir) {
		return core.Fail(coreerr.E(callerTrustedManifestPublicKeys, "symlinked trusted keys directory rejected: "+keyDir, nil))
	}

	entriesResult := core.ReadDir(core.DirFS(keyDir), ".")
	if !entriesResult.OK && core.IsNotExist(resultCause(entriesResult).(error)) {
		return core.Ok([]ed25519.PublicKey(nil))
	}
	if !entriesResult.OK {
		return core.Fail(coreerr.E(callerTrustedManifestPublicKeys, "read trusted keys directory failed", resultCause(entriesResult).(error)))
	}
	return trustedManifestKeysFromEntries(keyDir, entriesResult.Value.([]core.FsDirEntry))
}

func trustedManifestKeysFromEntries(keyDir string, entries []core.FsDirEntry) core.Result {
	keys := make([]ed25519.PublicKey, 0, len(entries))
	for _, entry := range entries {
		entryResult := trustedManifestPublicKeyFromEntry(keyDir, entry)
		if !entryResult.OK {
			return entryResult
		}
		trustedEntry := entryResult.Value.(trustedManifestPublicKeyEntry)
		if trustedEntry.Found {
			keys = append(keys, trustedEntry.Key)
		}
	}
	return core.Ok(dedupeManifestKeys(keys))
}

type trustedManifestPublicKeyEntry struct {
	Key   ed25519.PublicKey
	Found bool
}

func trustedManifestPublicKeyFromEntry(keyDir string, entry core.FsDirEntry) core.Result {
	if entry.IsDir() || !core.HasSuffix(entry.Name(), ".pub") {
		return core.Ok(trustedManifestPublicKeyEntry{})
	}

	entryPath := core.PathJoin(keyDir, entry.Name())
	if isSymlinkedLocalPath(entryPath) {
		return core.Fail(coreerr.E(callerTrustedManifestPublicKeys, "symlinked trusted key rejected: "+entry.Name(), nil))
	}
	bodyResult := core.ReadFSFile(core.DirFS(keyDir), entry.Name())
	if !bodyResult.OK {
		return core.Fail(coreerr.E(callerTrustedManifestPublicKeys, "read trusted key file failed: "+entry.Name(), resultCause(bodyResult).(error)))
	}
	pubResult := parseManifestPublicKey(core.Trim(string(bodyResult.Value.([]byte))))
	if !pubResult.OK {
		return core.Fail(coreerr.E(callerTrustedManifestPublicKeys, errDecodeTrustedKeyFailed+": "+entry.Name(), resultCause(pubResult).(error)))
	}
	return core.Ok(trustedManifestPublicKeyEntry{Key: pubResult.Value.(ed25519.PublicKey), Found: true})
}

func trustedManifestVerificationKeys() core.Result {
	if _, ok := core.LookupEnv("CORE_MANIFEST_TRUST_KEYS"); ok {
		return trustedManifestTrustedEnvKeys()
	}

	return trustedManifestPublicKeys()
}

func splitManifestTrustedKeys(raw string) []string {
	var out []string
	start := -1
	for i, r := range raw {
		if isManifestTrustKeySeparator(r) {
			if start >= 0 {
				out = append(out, raw[start:i])
				start = -1
			}
			continue
		}
		if start < 0 {
			start = i
		}
	}
	if start >= 0 {
		out = append(out, raw[start:])
	}
	return out
}

func isManifestTrustKeySeparator(r rune) bool {
	return r == ',' || r == ';' || r == '\n' || r == '\t' || r == ' ' || r == '\r'
}

func parseManifestPublicKey(raw string) core.Result {
	trimmed := core.Trim(raw)
	if trimmed == "" {
		return core.Fail(coreerr.E(callerParseManifestPublicKey, "empty manifest public key", nil))
	}
	pubResult := core.HexDecode(trimmed)
	if !pubResult.OK {
		return core.Fail(coreerr.E(callerParseManifestPublicKey, "decode manifest public key failed", resultCause(pubResult).(error)))
	}
	pub := pubResult.Value.([]byte)
	if len(pub) != ed25519.PublicKeySize {
		return core.Fail(coreerr.E(callerParseManifestPublicKey, "manifest public key has invalid size", nil))
	}
	return core.Ok(ed25519.PublicKey(pub))
}

func dedupeManifestKeys(keys []ed25519.PublicKey) []ed25519.PublicKey {
	seen := make(map[string]struct{}, len(keys))
	out := make([]ed25519.PublicKey, 0, len(keys))
	for _, key := range keys {
		if len(key) != ed25519.PublicKeySize {
			continue
		}
		serialized := string(key)
		if _, ok := seen[serialized]; ok {
			continue
		}
		out = append(out, key)
		seen[serialized] = struct{}{}
	}
	return out
}

func missingOrEmptyStringField(raw map[string]any, key string, current string) bool {
	if core.Trim(current) == "" {
		return true
	}
	rawValue, ok := raw[key]
	if !ok {
		return true
	}
	s, ok := rawValue.(string)
	return !ok || core.Trim(s) == ""
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

func firstLDFlags(values ...buildmanifestldflags) buildmanifestldflags {
	for _, value := range values {
		if len(value) > 0 {
			return append(buildmanifestldflags(nil), value...)
		}
	}
	return nil
}
