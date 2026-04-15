package config

import (
	"testing"

	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

func TestManifest_LoadManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/pkg/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\nlicence: EUPL-1.2\nsign: signed-manifest\nsign_key: pubkey\n"

	var pkg PackageManifest
	err := LoadManifest(m, "/pkg/.core/manifest.yaml", &pkg)
	assert.NoError(t, err)
	assert.Equal(t, "go-io", pkg.Code)
	assert.Equal(t, "Core I/O", pkg.Name)
	assert.Equal(t, "0.3.0", pkg.Version)
	assert.Equal(t, "EUPL-1.2", pkg.Licence)
}

func TestManifest_LoadManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	var pkg PackageManifest
	err := LoadManifest(m, "/nonexistent.yaml", &pkg)
	assert.Error(t, err)
}

func TestManifest_LoadManifest_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/bad.yaml"] = "this is: [not: valid: yaml"

	var pkg PackageManifest
	err := LoadManifest(m, "/bad.yaml", &pkg)
	assert.Error(t, err)
}

func TestManifest_LoadManifest_Build_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/build.yaml"] = "name: core\noutput: dist\ncgo: false\ntargets:\n  - os: linux\n    arch: amd64\n  - os: darwin\n    arch: arm64\n"

	var build BuildManifest
	err := LoadManifest(m, "/.core/build.yaml", &build)
	assert.NoError(t, err)
	assert.Equal(t, "core", build.Name)
	assert.Equal(t, "dist", build.Output)
	assert.Len(t, build.Targets, 2)
	assert.Equal(t, "linux", build.Targets[0].OS)
	assert.Equal(t, "amd64", build.Targets[0].Arch)
}

func TestManifest_LoadManifest_Build_ShorthandTargets_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/build.yaml"] = "name: core\noutput: dist\ntargets:\n  - linux/amd64\n  - darwin/arm64\nsign:\n  enabled: true\n  gpg:\n    key: $GPG_KEY_ID\n  macos:\n    identity: 'Developer ID Application: Example'\n    notarize: false\nsdk:\n  spec: openapi.yaml\n  languages:\n    - typescript\n    - go\n  output: sdk/\n  diff: true\n"

	var build BuildManifest
	err := LoadManifest(m, "/.core/build.yaml", &build)
	assert.NoError(t, err)
	assert.Len(t, build.Targets, 2)
	assert.Equal(t, "linux", build.Targets[0].OS)
	assert.Equal(t, "amd64", build.Targets[0].Arch)
	assert.True(t, build.Signing.Enabled)
	assert.Equal(t, "$GPG_KEY_ID", build.Signing.GPG.Key)
	assert.Equal(t, "Developer ID Application: Example", build.Signing.MacOS.Identity)
	assert.True(t, build.SDK.Diff)
	assert.Equal(t, "openapi.yaml", build.SDK.Spec)
	assert.Equal(t, []string{"typescript", "go"}, build.SDK.Languages)
}

func TestManifest_LoadManifest_View_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/view.yaml"] = "code: photo-browser\nname: Photo Browser\nsign: signed-view\npermissions:\n  clipboard: true\n  filesystem: true\n"

	var view ViewManifest
	err := LoadManifest(m, "/.core/view.yaml", &view)
	assert.NoError(t, err)
	assert.Equal(t, "photo-browser", view.Code)
	assert.True(t, view.Permissions.Clipboard)
	assert.True(t, view.Permissions.Filesystem)
}

func TestManifest_LoadManifest_Test_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/test.yaml"] = "version: 1\ncommands:\n  - name: unit\n    run: vendor/bin/pest --parallel\n  - name: types\n    run: vendor/bin/phpstan analyse\nenv:\n  APP_ENV: testing\n  DB_CONNECTION: sqlite\n"

	var test TestManifest
	err := LoadManifest(m, "/.core/test.yaml", &test)
	assert.NoError(t, err)
	assert.Equal(t, 1, test.Version)
	assert.Len(t, test.Commands, 2)
	assert.Equal(t, "unit", test.Commands[0].Name)
	assert.Equal(t, "vendor/bin/pest --parallel", test.Commands[0].Run)
	assert.Equal(t, "testing", test.Env["APP_ENV"])
}

func TestManifest_LoadManifest_Run_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/run.yaml"] = "version: 1\nservices:\n  - name: database\n    image: postgres:16\n    port: 5432\n    env:\n      POSTGRES_DB: core_dev\ndev:\n  command: php artisan serve\n  port: 8000\n  watch:\n    - app/\n    - resources/\nenv:\n  APP_ENV: local\n"

	var run RunManifest
	err := LoadManifest(m, "/.core/run.yaml", &run)
	assert.NoError(t, err)
	assert.Equal(t, 1, run.Version)
	assert.Len(t, run.Services, 1)
	assert.Equal(t, "database", run.Services[0].Name)
	assert.Equal(t, 5432, run.Services[0].Port)
	assert.Equal(t, "core_dev", run.Services[0].Env["POSTGRES_DB"])
	assert.Equal(t, "php artisan serve", run.Dev.Command)
	assert.Equal(t, 8000, run.Dev.Port)
	assert.Contains(t, run.Dev.Watch, "app/")
}

func TestManifest_LoadManifest_Repos_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/Code/.core/repos.yaml"] = "org: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n    branch: dev\n    type: lib\n    depends:\n      - go-io\n  - path: core/config\n    remote: ssh://forge.example/core/config.git\n    branch: dev\n"

	var repos ReposManifest
	err := LoadManifest(m, "/Code/.core/repos.yaml", &repos)
	assert.NoError(t, err)
	assert.Equal(t, "host-uk", repos.Org)
	assert.Len(t, repos.Repos, 2)
	assert.Equal(t, "core/go", repos.Repos[0].Path)
	assert.Equal(t, "dev", repos.Repos[0].Branch)
	assert.Equal(t, "lib", repos.Repos[0].Type)
	assert.Contains(t, repos.Repos[0].Depends, "go-io")
}

func TestManifest_LoadManifest_Repos_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	var repos ReposManifest
	err := LoadManifest(m, "/missing/repos.yaml", &repos)
	assert.Error(t, err)
}

func TestManifest_LoadManifest_Release_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/release.yaml"] = "archive:\n  format: tar.gz\n  include:\n    - LICENSE.txt\n    - README.md\nchecksums: true\ngithub:\n  draft: false\n  prerelease: false\nchangelog:\n  include:\n    - feat\n    - fix\n"

	var rel ReleaseManifest
	err := LoadManifest(m, "/.core/release.yaml", &rel)
	assert.NoError(t, err)
	assert.Equal(t, "tar.gz", rel.Archive.Format)
	assert.Contains(t, rel.Archive.Include, "LICENSE.txt")
	assert.True(t, rel.Checksums)
	assert.False(t, rel.GitHub.Draft)
	assert.Contains(t, rel.Changelog.Include, "feat")
}

func TestManifest_LoadManifest_Agent_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/home/.core/agent.yaml"] = "daemon:\n  enabled: true\n  watch:\n    - ~/Code/core/\n  schedule:\n    - cron: '*/5 * * * *'\n      action: health.check\n  mcp:\n    port: 0\n  api:\n    port: 8099\n    bind: 127.0.0.1\nagents:\n  codex:\n    total: 2\n  claude:\n    total: 1\n"

	var agent AgentManifest
	err := LoadManifest(m, "/home/.core/agent.yaml", &agent)
	assert.NoError(t, err)
	assert.True(t, agent.Daemon.Enabled)
	assert.Equal(t, "health.check", agent.Daemon.Schedule[0].Action)
	assert.Equal(t, 8099, agent.Daemon.API.Port)
	assert.Equal(t, 2, agent.Agents["codex"].Total)
}

func TestManifest_LoadManifest_Zone_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/home/.core/zone.yaml"] = "zone:\n  name: snider\n  identity: '@snider@lthn'\n  chain:\n    mode: thin\n    daemon: localhost:36941\n  network:\n    wireguard:\n      interface: wg-lthn\n      listen: 51820\n  services:\n    vpn:\n      enabled: true\n      price: 0.001\n      capacity: 100\n    dns:\n      enabled: true\n    compute:\n      enabled: true\n      models:\n        - lem-1b\n        - lem-4b\n  staking:\n    amount: 1000\n    tier: trusted\n"

	var zone ZoneManifest
	err := LoadManifest(m, "/home/.core/zone.yaml", &zone)
	assert.NoError(t, err)
	assert.Equal(t, "snider", zone.Zone.Name)
	assert.Equal(t, "thin", zone.Zone.Chain.Mode)
	assert.Equal(t, "wg-lthn", zone.Zone.Network.WireGuard.Interface)
	assert.Equal(t, 100, zone.Zone.Services.VPN.Capacity)
	assert.Contains(t, zone.Zone.Services.Compute.Models, "lem-4b")
}

func TestManifest_LoadManifest_Schema_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/build.yaml"] = "targets: 42\n"

	var build BuildManifest
	err := LoadManifest(m, "/.core/build.yaml", &build)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation failed")
}

func TestManifest_KnownFiles_Good(t *testing.T) {
	// The constants are single-source-of-truth names; KnownFiles must contain
	// every canonical project-level file and not duplicate any.
	assert.Contains(t, KnownFiles, FileConfig)
	assert.Contains(t, KnownFiles, FileBuild)
	assert.Contains(t, KnownFiles, FileTest)
	assert.Contains(t, KnownFiles, FileRun)
	assert.Contains(t, KnownFiles, FileRelease)
	assert.Contains(t, KnownFiles, FileView)
	assert.Contains(t, KnownFiles, FileManifest)
	assert.Contains(t, KnownFiles, FileWorkspace)
	assert.Contains(t, KnownFiles, FileRepos)
	assert.Contains(t, KnownFiles, FileIDE)
	assert.Contains(t, KnownFiles, FilePHP)
	assert.Equal(t, ".core", Directory)

	// User-level files have constants but are not part of project discovery.
	assert.Equal(t, "agent.yaml", FileAgent)
	assert.Equal(t, "zone.yaml", FileZone)
	assert.Equal(t, "ide.yaml", FileIDE)
	assert.Equal(t, "php.yaml", FilePHP)

	seen := map[string]struct{}{}
	for _, name := range KnownFiles {
		_, dup := seen[name]
		assert.False(t, dup, "duplicate known file: %s", name)
		seen[name] = struct{}{}
	}
}
