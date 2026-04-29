package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"

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
func (v *ViewVersion) UnmarshalYAML(node *yaml.Node) configError {
	switch node.Kind {
	case yaml.ScalarNode:
		var asString string
		if err := node.Decode(&asString); err == nil {
			*v = ViewVersion(asString)
			return nil
		}
		var asInt int
		if err := node.Decode(&asInt); err == nil {
			*v = ViewVersion(core.Sprintf("%d", asInt))
			return nil
		}
	case yaml.AliasNode:
		return node.Decode(v)
	}
	return coreerr.E("config.ViewVersion.UnmarshalYAML", "invalid view manifest version", nil)
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
func (t *BuildTarget) UnmarshalYAML(value *yaml.Node) configError {
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
		parts := core.SplitN(raw, "/", 2)
		if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
			return coreerr.E("config.BuildTarget.UnmarshalYAML", "invalid target shorthand: "+raw, nil)
		}
		*t = BuildTarget{OS: parts[0], Arch: parts[1]}
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
	Targets []BuildTarget        `yaml:"targets"`
	Signing buildManifestSigning `yaml:"sign"`
	SDK     buildManifestSDK     `yaml:"sdk"`
	Name    string               `yaml:"name"`
	Main    string               `yaml:"main"`
	Binary  string               `yaml:"binary"`
	Output  string               `yaml:"output"`
	Flags   []string             `yaml:"flags"`
	LDFlags buildmanifestldflags `yaml:"ldflags"`
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
	LDFlags buildmanifestldflags `yaml:"ldflags"`
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

func (l *buildmanifestldflags) UnmarshalYAML(value *yaml.Node) configError {
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
	case yaml.AliasNode:
		var values []string
		if err := value.Decode(&values); err != nil {
			return err
		}
		*l = append([]string(nil), values...)
		return nil
	case yaml.MappingNode:
		return coreerr.E("config.buildmanifestldflags.UnmarshalYAML", "unsupported ldflags mapping", nil)
	default:
		*l = nil
		return nil
	}
}

func (l buildmanifestldflags) String() string {
	return core.Join(" ", l...)
}

// UnmarshalYAML accepts both the legacy flat build schema and the nested
// RFC shape with project/build sections.
func (m *BuildManifest) UnmarshalYAML(value *yaml.Node) configError {
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
	m.LDFlags = core.Join(" ", m.Build.LDFlags...)
	m.CGO = m.Build.CGO
	m.Env = raw.Env
	return nil
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
func LoadManifest(m coreio.Medium, path string, out any) configError {
	content, err := m.Read(path)
	if err != nil {
		return coreerr.E(callerLoadManifest, "failed to read manifest: "+path, err)
	}
	var raw map[string]any
	if err := yaml.Unmarshal([]byte(content), &raw); err != nil {
		return coreerr.E(callerLoadManifest, "failed to parse manifest: "+path, err)
	}
	if err := validateSchema(path, raw); err != nil {
		return err
	}
	if err := yaml.Unmarshal([]byte(content), out); err != nil {
		return coreerr.E(callerLoadManifest, "failed to decode manifest: "+path, err)
	}
	if err := validateManifest(path, out, raw); err != nil {
		return err
	}
	return nil
}

func validateManifest(path string, out any, raw map[string]any) configError {
	switch core.PathBase(path) {
	case FileView:
		view, ok := out.(*ViewManifest)
		if !ok {
			return nil
		}
		if err := validateLoadedViewManifest(path, view, raw); err != nil {
			return err
		}
	case FileManifest:
		pkg, ok := out.(*PackageManifest)
		if !ok {
			return nil
		}
		if err := verifyLoadedPackageManifest(path, pkg, raw); err != nil {
			return err
		}
	}
	return nil
}

func validateLoadedViewManifest(path string, view *ViewManifest, raw map[string]any) configError {
	if missingOrEmptyStringField(raw, "sign", view.Sign) {
		return coreerr.E(callerLoadManifest, "unsigned view manifest rejected: "+path, nil)
	}
	if err := ValidateViewManifestSignature(view); err != nil {
		msg := err.Error()
		switch {
		case core.Contains(msg, "not ed25519-sized"):
			return coreerr.E(callerLoadManifest, "view manifest signature is not ed25519-sized: "+path, nil)
		case core.Contains(msg, "unsigned"):
			return coreerr.E(callerLoadManifest, "unsigned view manifest rejected: "+path, nil)
		default:
			return coreerr.E(callerLoadManifest, "invalid view manifest signature: "+path, err)
		}
	}
	return nil
}

func verifyLoadedPackageManifest(path string, pkg *PackageManifest, raw map[string]any) configError {
	if missingOrEmptyStringField(raw, "sign", pkg.Sign) {
		return coreerr.E(callerLoadManifest, "unsigned package manifest rejected: "+path, nil)
	}
	if missingOrEmptyStringField(raw, "sign_key", pkg.SignKey) {
		return coreerr.E(callerLoadManifest, "missing package sign_key: "+path, nil)
	}
	if err := VerifyPackageManifest(pkg); err != nil {
		msg := err.Error()
		switch {
		case core.Contains(msg, "missing package sign_key"):
			return coreerr.E(callerLoadManifest, "missing package sign_key: "+path, nil)
		case core.Contains(msg, "not an ed25519 public key"):
			return coreerr.E(callerLoadManifest, "package sign_key is not an ed25519 public key: "+path, nil)
		case core.Contains(msg, "not trusted"):
			return coreerr.E(callerLoadManifest, "package sign_key is not trusted: "+path, nil)
		case core.Contains(msg, "not ed25519-sized"):
			return coreerr.E(callerLoadManifest, "package manifest signature is not ed25519-sized: "+path, nil)
		case core.Contains(msg, "signature mismatch"):
			return coreerr.E(callerLoadManifest, "package manifest signature mismatch: "+path, nil)
		case core.Contains(msg, errCanonicalMarshalFailed):
			return coreerr.E(callerLoadManifest, errCanonicalMarshalFailed+": "+path, err)
		case core.Contains(msg, "decode package sign_key failed"):
			return coreerr.E(callerLoadManifest, "decode package sign_key failed: "+path, err)
		default:
			return coreerr.E(callerLoadManifest, "invalid package manifest signature: "+path, err)
		}
	}
	return nil
}

func manifestKeyTrusted(candidate ed25519.PublicKey, trusted []ed25519.PublicKey) bool {
	for _, key := range trusted {
		if len(key) != ed25519.PublicKeySize {
			continue
		}
		if core.Lower(hex.EncodeToString(candidate)) == core.Lower(hex.EncodeToString(key)) {
			return true
		}
	}
	return false
}

func trustedManifestTrustedEnvKeys() ([]ed25519.PublicKey, error) {
	fromEnv := core.Trim(core.Env("CORE_MANIFEST_TRUST_KEYS"))
	if fromEnv == "" {
		return nil, nil
	}
	return parseTrustedManifestKeyList(fromEnv)
}

func decodeManifestSignature(value string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(core.Trim(value))
}

// CanonicalViewManifestBytes returns the RFC canonical view manifest body with
// the sign field cleared so callers can sign or verify it consistently.
//
//	body, _ := config.CanonicalViewManifestBytes(&view)
func CanonicalViewManifestBytes(view *ViewManifest) ([]byte, error) {
	return viewManifestBytes(view)
}

// ValidateViewManifestSignature checks only that view.yaml carries a base64
// ed25519-sized signature. Trust-root verification belongs to the caller.
//
//	if err := config.ValidateViewManifestSignature(&view); err != nil { ... }
func ValidateViewManifestSignature(view *ViewManifest) configError {
	if view == nil || core.Trim(view.Sign) == "" {
		return coreerr.E(callerValidateViewManifestSignature, "unsigned view manifest rejected", nil)
	}
	sig, err := decodeManifestSignature(view.Sign)
	if err != nil {
		return coreerr.E(callerValidateViewManifestSignature, "invalid view manifest signature", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return coreerr.E(callerValidateViewManifestSignature, "view manifest signature is not ed25519-sized", nil)
	}
	return nil
}

// VerifyViewManifestSignature verifies a signed view manifest against the
// caller-supplied ed25519 public key.
//
//	if err := config.VerifyViewManifestSignature(&view, pub); err != nil { ... }
func VerifyViewManifestSignature(view *ViewManifest, publicKey ed25519.PublicKey) configError {
	if err := ValidateViewManifestSignature(view); err != nil {
		return err
	}
	if len(publicKey) != ed25519.PublicKeySize {
		return coreerr.E(callerVerifyViewManifestSignature, "view manifest public key is not an ed25519 public key", nil)
	}
	body, err := CanonicalViewManifestBytes(view)
	if err != nil {
		return coreerr.E(callerVerifyViewManifestSignature, errCanonicalMarshalFailed, err)
	}
	sig, err := decodeManifestSignature(view.Sign)
	if err != nil {
		return coreerr.E(callerVerifyViewManifestSignature, "invalid view manifest signature", err)
	}
	if !ed25519.Verify(publicKey, body, sig) {
		return coreerr.E(callerVerifyViewManifestSignature, "view manifest signature mismatch", nil)
	}
	return nil
}

// SignViewManifest signs view.yaml in place using the RFC canonical body and
// stores the resulting base64 signature in Sign.
//
//	_ = config.SignViewManifest(&view, priv)
func SignViewManifest(view *ViewManifest, privateKey ed25519.PrivateKey) configError {
	if view == nil {
		return coreerr.E(callerSignViewManifest, "nil view manifest", nil)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return coreerr.E(callerSignViewManifest, "view manifest private key is not an ed25519 private key", nil)
	}
	body, err := CanonicalViewManifestBytes(view)
	if err != nil {
		return coreerr.E(callerSignViewManifest, errCanonicalMarshalFailed, err)
	}
	view.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, body))
	return nil
}

func viewManifestBytes(view *ViewManifest) ([]byte, error) {
	if view == nil {
		return yaml.Marshal(nil)
	}
	tmp := *view
	tmp.Sign = ""
	return yaml.Marshal(&tmp)
}

// CanonicalPackageManifestBytes returns the RFC canonical package manifest body
// with the sign field cleared so callers can sign or verify it consistently.
//
//	body, _ := config.CanonicalPackageManifestBytes(&pkg)
func CanonicalPackageManifestBytes(pkg *PackageManifest) ([]byte, error) {
	return packageManifestBytes(pkg)
}

// SignPackageManifest signs manifest.yaml in place and ensures SignKey matches
// the supplied ed25519 private key's public half.
//
//	_ = config.SignPackageManifest(&pkg, priv)
func SignPackageManifest(pkg *PackageManifest, privateKey ed25519.PrivateKey) configError {
	if pkg == nil {
		return coreerr.E(callerSignPackageManifest, "nil package manifest", nil)
	}
	if len(privateKey) != ed25519.PrivateKeySize {
		return coreerr.E(callerSignPackageManifest, "package manifest private key is not an ed25519 private key", nil)
	}
	publicKey, ok := privateKey.Public().(ed25519.PublicKey)
	if !ok || len(publicKey) != ed25519.PublicKeySize {
		return coreerr.E(callerSignPackageManifest, "derive package manifest public key failed", nil)
	}
	pkg.SignKey = hex.EncodeToString(publicKey)
	body, err := CanonicalPackageManifestBytes(pkg)
	if err != nil {
		return coreerr.E(callerSignPackageManifest, errCanonicalMarshalFailed, err)
	}
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(privateKey, body))
	return nil
}

func packageManifestBytes(pkg *PackageManifest) ([]byte, error) {
	if pkg == nil {
		return yaml.Marshal(nil)
	}
	tmp := *pkg
	tmp.Sign = ""
	return yaml.Marshal(&tmp)
}

// VerifyPackageManifest verifies manifest.yaml against its embedded sign_key
// and the optional trust roots from CORE_MANIFEST_TRUST_KEYS / ~/.core/keys.
//
//	if err := config.VerifyPackageManifest(&pkg); err != nil { ... }
func VerifyPackageManifest(pkg *PackageManifest) configError {
	if pkg == nil || core.Trim(pkg.Sign) == "" {
		return coreerr.E(callerVerifyPackageManifest, "unsigned package manifest rejected", nil)
	}
	if core.Trim(pkg.SignKey) == "" {
		return coreerr.E(callerVerifyPackageManifest, "missing package sign_key", nil)
	}

	pub, err := hex.DecodeString(core.Trim(pkg.SignKey))
	if err != nil {
		return coreerr.E(callerVerifyPackageManifest, "decode package sign_key failed", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return coreerr.E(callerVerifyPackageManifest, "package sign_key is not an ed25519 public key", nil)
	}

	trustedKeys, err := trustedManifestVerificationKeys()
	if err != nil {
		return coreerr.E(callerVerifyPackageManifest, "load trusted manifest public keys failed", err)
	}
	if len(trustedKeys) > 0 && !manifestKeyTrusted(ed25519.PublicKey(pub), trustedKeys) {
		return coreerr.E(callerVerifyPackageManifest, "package sign_key is not trusted", nil)
	}

	sig, err := decodeManifestSignature(pkg.Sign)
	if err != nil {
		return coreerr.E(callerVerifyPackageManifest, "invalid package manifest signature", err)
	}
	if len(sig) != ed25519.SignatureSize {
		return coreerr.E(callerVerifyPackageManifest, "package manifest signature is not ed25519-sized", nil)
	}

	body, err := CanonicalPackageManifestBytes(pkg)
	if err != nil {
		return coreerr.E(callerVerifyPackageManifest, errCanonicalMarshalFailed, err)
	}
	if !ed25519.Verify(ed25519.PublicKey(pub), body, sig) {
		return coreerr.E(callerVerifyPackageManifest, "package manifest signature mismatch", nil)
	}
	return nil
}

// TrustedManifestPublicKeys returns the deduplicated trust roots discovered
// from CORE_MANIFEST_TRUST_KEYS or ~/.core/keys/*.pub.
//
//	keys, _ := config.TrustedManifestPublicKeys()
func TrustedManifestPublicKeys() ([]ed25519.PublicKey, error) {
	return trustedManifestPublicKeys()
}

func trustedManifestPublicKeys() ([]ed25519.PublicKey, error) {
	if fromEnv := core.Trim(core.Env("CORE_MANIFEST_TRUST_KEYS")); fromEnv != "" {
		return parseTrustedManifestKeyList(fromEnv)
	}

	return trustedManifestPublicKeysFromDisk(manifestHomeDir())
}

func parseTrustedManifestKeyList(raw string) ([]ed25519.PublicKey, error) {
	var keys []ed25519.PublicKey
	for _, item := range splitManifestTrustedKeys(raw) {
		pub, err := parseManifestPublicKey(item)
		if err != nil {
			return nil, coreerr.E(callerTrustedManifestPublicKeys, errDecodeTrustedKeyFailed, err)
		}
		keys = append(keys, pub)
	}
	return dedupeManifestKeys(keys), nil
}

func trustedManifestPublicKeysFromDisk(home string) ([]ed25519.PublicKey, error) {
	if home == "" {
		return nil, nil
	}

	coreDir := core.PathJoin(home, ".core")
	keyDir := core.PathJoin(coreDir, "keys")
	if isSymlinkedCoreDir(coreio.Local, coreDir) {
		return nil, coreerr.E(callerTrustedManifestPublicKeys, "symlinked .core directory rejected: "+coreDir, nil)
	}
	if isSymlinkedLocalPath(keyDir) {
		return nil, coreerr.E(callerTrustedManifestPublicKeys, "symlinked trusted keys directory rejected: "+keyDir, nil)
	}

	entriesResult := core.ReadDir(core.DirFS(keyDir), ".")
	if !entriesResult.OK && core.IsNotExist(resultError(entriesResult)) {
		return nil, nil
	}
	if !entriesResult.OK {
		return nil, coreerr.E(callerTrustedManifestPublicKeys, "read trusted keys directory failed", resultError(entriesResult))
	}
	return trustedManifestKeysFromEntries(keyDir, entriesResult.Value.([]core.FsDirEntry))
}

func trustedManifestKeysFromEntries(keyDir string, entries []core.FsDirEntry) ([]ed25519.PublicKey, error) {
	keys := make([]ed25519.PublicKey, 0, len(entries))
	for _, entry := range entries {
		pub, ok, err := trustedManifestPublicKeyFromEntry(keyDir, entry)
		if err != nil {
			return nil, err
		}
		if ok {
			keys = append(keys, pub)
		}
	}
	return dedupeManifestKeys(keys), nil
}

func trustedManifestPublicKeyFromEntry(keyDir string, entry core.FsDirEntry) (ed25519.PublicKey, bool, error) {
	if entry.IsDir() || !core.HasSuffix(entry.Name(), ".pub") {
		return nil, false, nil
	}

	entryPath := core.PathJoin(keyDir, entry.Name())
	if isSymlinkedLocalPath(entryPath) {
		return nil, false, coreerr.E(callerTrustedManifestPublicKeys, "symlinked trusted key rejected: "+entry.Name(), nil)
	}
	bodyResult := core.ReadFSFile(core.DirFS(keyDir), entry.Name())
	if !bodyResult.OK {
		return nil, false, coreerr.E(callerTrustedManifestPublicKeys, "read trusted key file failed: "+entry.Name(), resultError(bodyResult))
	}
	pub, err := parseManifestPublicKey(core.Trim(string(bodyResult.Value.([]byte))))
	if err != nil {
		return nil, false, coreerr.E(callerTrustedManifestPublicKeys, errDecodeTrustedKeyFailed+": "+entry.Name(), err)
	}
	return pub, true, nil
}

func trustedManifestVerificationKeys() ([]ed25519.PublicKey, error) {
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

func parseManifestPublicKey(raw string) (ed25519.PublicKey, error) {
	trimmed := core.Trim(raw)
	if trimmed == "" {
		return nil, coreerr.E(callerParseManifestPublicKey, "empty manifest public key", nil)
	}
	pub, err := hex.DecodeString(trimmed)
	if err != nil {
		return nil, coreerr.E(callerParseManifestPublicKey, "decode manifest public key failed", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		return nil, coreerr.E(callerParseManifestPublicKey, "manifest public key has invalid size", nil)
	}
	return ed25519.PublicKey(pub), nil
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
