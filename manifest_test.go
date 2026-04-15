package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

func setManifestTrustKeys(t *testing.T, keys ...string) {
	t.Helper()
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", strings.Join(keys, ","))
}

func TestManifest_SplitManifestTrustedKeys_Good(t *testing.T) {
	got := splitManifestTrustedKeys("a,b;c d\te\nf")
	assert.Equal(t, []string{"a", "b", "c", "d", "e", "f"}, got)
}

func TestManifest_SplitManifestTrustedKeys_Bad(t *testing.T) {
	got := splitManifestTrustedKeys("")
	assert.Empty(t, got)
}

func TestManifest_SplitManifestTrustedKeys_Ugly(t *testing.T) {
	got := splitManifestTrustedKeys("   ")
	assert.Empty(t, got)
}

func TestManifest_ParseManifestPublicKey_Good(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	got, err := parseManifestPublicKey(hex.EncodeToString(pub))
	assert.NoError(t, err)
	assert.Equal(t, hex.EncodeToString(pub), hex.EncodeToString(got))
}

func TestManifest_ParseManifestPublicKey_Bad(t *testing.T) {
	_, err := parseManifestPublicKey("not-hex")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode manifest public key failed")
}

func TestManifest_ParseManifestPublicKey_Ugly(t *testing.T) {
	_, err := parseManifestPublicKey("   ")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty manifest public key")
}

func TestManifest_DedupeManifestKeys_Good(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	got := dedupeManifestKeys([]ed25519.PublicKey{pub, pub})
	assert.Equal(t, []ed25519.PublicKey{pub}, got)
}

func TestManifest_DedupeManifestKeys_Bad(t *testing.T) {
	out := dedupeManifestKeys(nil)
	assert.Empty(t, out)
}

func TestManifest_DedupeManifestKeys_Ugly(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	invalid := ed25519.PublicKey("short")
	out := dedupeManifestKeys([]ed25519.PublicKey{invalid, invalid, pub})
	assert.Equal(t, []ed25519.PublicKey{pub}, out)
}

func TestManifest_MissingOrEmptyStringField_Good(t *testing.T) {
	raw := map[string]any{"sign": "abc"}
	assert.False(t, missingOrEmptyStringField(raw, "sign", "abc"))
}

func TestManifest_MissingOrEmptyStringField_Bad(t *testing.T) {
	raw := map[string]any{}
	assert.True(t, missingOrEmptyStringField(raw, "sign", "abc"))
}

func TestManifest_MissingOrEmptyStringField_Ugly(t *testing.T) {
	raw := map[string]any{"sign": ""}
	assert.True(t, missingOrEmptyStringField(raw, "sign", "abc"))
	raw["sign"] = "   "
	assert.True(t, missingOrEmptyStringField(raw, "sign", "abc"))
}

func TestManifest_TrustedManifestPublicKeys_Good(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	got, err := trustedManifestPublicKeys()
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, pub, got[0])
}

func TestManifest_TrustedManifestPublicKeys_Bad(t *testing.T) {
	setManifestTrustKeys(t, "not-hex")
	_, err := trustedManifestPublicKeys()
	assert.Error(t, err)
}

func TestManifest_TrustedManifestPublicKeys_Ugly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("DIR_HOME", home)
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")

	keysDir := filepath.Join(home, ".core", "keys")
	assert.NoError(t, os.MkdirAll(keysDir, 0o755))

	pub, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	assert.NoError(t, os.WriteFile(filepath.Join(keysDir, "trusted.pub"), []byte(fmt.Sprintf("%x\n", pub)), 0o644))

	got, err := trustedManifestPublicKeys()
	assert.NoError(t, err)
	assert.Len(t, got, 1)
}

func TestManifest_TrustedManifestPublicKeys_SymlinkedCore_Bad(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is not portable on Windows in this environment")
	}

	home := core.Env("DIR_HOME")
	if home == "" {
		t.Skip("DIR_HOME is empty in this environment")
	}

	coreDir := filepath.Join(home, ".core")
	if _, err := os.Lstat(coreDir); err == nil {
		t.Skip("DIR_HOME/.core already exists in this environment")
	}

	realCore := filepath.Join(t.TempDir(), "real-core")
	assert.NoError(t, os.MkdirAll(realCore, 0o755))
	assert.NoError(t, os.Symlink(realCore, coreDir))
	t.Cleanup(func() { _ = os.Remove(coreDir) })

	_, err := trustedManifestPublicKeys()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symlinked .core directory rejected")
}

func TestManifest_TrustedManifestPublicKeysExported_Good(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	got, err := TrustedManifestPublicKeys()
	assert.NoError(t, err)
	assert.Len(t, got, 1)
	assert.Equal(t, pub, got[0])
}

func TestManifest_ViewSignatureHelpers_Good(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	view := &ViewManifest{
		Code:    "photo-browser",
		Name:    "Photo Browser",
		Version: ViewVersion("0.1.0"),
		Layout:  "HLCRF",
		Slots: map[string]any{
			"C": "photo-grid",
		},
	}

	body, err := CanonicalViewManifestBytes(view)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "sign: \"\"")

	err = SignViewManifest(view, priv)
	assert.NoError(t, err)
	assert.NotEmpty(t, view.Sign)
	assert.NoError(t, ValidateViewManifestSignature(view))
	assert.NoError(t, VerifyViewManifestSignature(view, pub))
}

func TestManifest_ViewSignatureHelpers_Bad(t *testing.T) {
	view := &ViewManifest{
		Code: "photo-browser",
		Name: "Photo Browser",
		Sign: "not-base64!!",
	}

	err := ValidateViewManifestSignature(view)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid view manifest signature")
}

func TestManifest_ViewSignatureHelpers_Ugly(t *testing.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	view := &ViewManifest{
		Code: "photo-browser",
		Name: "Photo Browser",
		Sign: base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
	}

	err = VerifyViewManifestSignature(view, pub[:ed25519.PublicKeySize-1])
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an ed25519 public key")
}

func TestManifest_PackageSignatureHelpers_Good(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        "Core I/O",
		Version:     "0.3.0",
		Description: "Mandatory I/O abstraction layer",
		Licence:     "EUPL-1.2",
	}

	body, err := CanonicalPackageManifestBytes(pkg)
	assert.NoError(t, err)
	assert.Contains(t, string(body), "sign: \"\"")
	assert.Contains(t, string(body), "sign_key: \"\"")

	err = SignPackageManifest(pkg, priv)
	assert.NoError(t, err)
	assert.NotEmpty(t, pkg.Sign)
	assert.NotEmpty(t, pkg.SignKey)
	assert.NoError(t, VerifyPackageManifest(pkg))
}

func TestManifest_PackageSignatureHelpers_Bad(t *testing.T) {
	pkg := &PackageManifest{
		Code:    "go-io",
		Name:    "Core I/O",
		Version: "0.3.0",
		SignKey: "not-hex",
		Sign:    base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
	}

	err := VerifyPackageManifest(pkg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode package sign_key failed")
}

func TestManifest_PackageSignatureHelpers_Ugly(t *testing.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        "Core I/O",
		Version:     "0.3.0",
		Description: "Mandatory I/O abstraction layer",
		Licence:     "EUPL-1.2",
	}

	err = SignPackageManifest(pkg, priv)
	assert.NoError(t, err)
	pkg.Description = "Tampered"

	err = VerifyPackageManifest(pkg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature mismatch")
}

func TestManifest_LoadManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	pub, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	signedPkg := &PackageManifest{
		Code:    "go-io",
		Name:    "Core I/O",
		Version: "0.3.0",
		Licence: "EUPL-1.2",
		SignKey: hex.EncodeToString(pub),
	}
	msg, err := packageManifestBytes(signedPkg)
	assert.NoError(t, err)
	signedPkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))
	m.Files["/pkg/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\nlicence: EUPL-1.2\nsign_key: " + signedPkg.SignKey + "\nsign: " + signedPkg.Sign + "\n"

	var pkg PackageManifest
	err = LoadManifest(m, "/pkg/.core/manifest.yaml", &pkg)
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

func TestManifest_LoadManifest_Build_LegacyFlat_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/build.yaml"] = "name: core\nmain: ./cmd/core\nbinary: core\noutput: dist\nflags:\n  - -trimpath\nldflags: -s -w\ncgo: false\ntargets:\n  - linux/amd64\n"

	var build BuildManifest
	err := LoadManifest(m, "/.core/build.yaml", &build)
	assert.NoError(t, err)
	assert.Equal(t, "core", build.Name)
	assert.Equal(t, "./cmd/core", build.Main)
	assert.Equal(t, "core", build.Binary)
	assert.Equal(t, "dist", build.Output)
	assert.Equal(t, []string{"-trimpath"}, build.Flags)
	assert.Equal(t, "-s -w", build.LDFlags)
	assert.False(t, build.CGO)
	assert.Len(t, build.Targets, 1)
}

func TestManifest_BuildTarget_UnmarshalYAML_Good(t *testing.T) {
	var target BuildTarget
	assert.NoError(t, target.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "linux/amd64"}))
	assert.Equal(t, BuildTarget{OS: "linux", Arch: "amd64"}, target)

	assert.NoError(t, target.UnmarshalYAML(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "os"},
			{Kind: yaml.ScalarNode, Value: "darwin"},
			{Kind: yaml.ScalarNode, Value: "arch"},
			{Kind: yaml.ScalarNode, Value: "arm64"},
		},
	}))
	assert.Equal(t, BuildTarget{OS: "darwin", Arch: "arm64"}, target)
}

func TestManifest_BuildTarget_UnmarshalYAML_Bad(t *testing.T) {
	var target BuildTarget

	err := target.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "linux"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid target shorthand")
}

func TestManifest_BuildTarget_UnmarshalYAML_Ugly(t *testing.T) {
	var target BuildTarget

	assert.NoError(t, target.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: ""}))
	assert.Equal(t, BuildTarget{}, target)
}

func TestManifest_BuildManifestLDFlags_String_Good(t *testing.T) {
	flags := buildManifestLDFlags{"-s", "-w"}
	assert.Equal(t, "-s -w", flags.String())
}

func TestManifest_BuildManifestLDFlags_UnmarshalYAML_Good(t *testing.T) {
	var flags buildManifestLDFlags

	assert.NoError(t, flags.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "-s -w"}))
	assert.Equal(t, buildManifestLDFlags{"-s -w"}, flags)

	assert.NoError(t, flags.UnmarshalYAML(&yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "-s"},
			{Kind: yaml.ScalarNode, Value: "-w"},
		},
	}))
	assert.Equal(t, buildManifestLDFlags{"-s", "-w"}, flags)
}

func TestManifest_BuildManifestLDFlags_UnmarshalYAML_Bad(t *testing.T) {
	var flags buildManifestLDFlags

	err := flags.UnmarshalYAML(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "0"},
			{Kind: yaml.ScalarNode, Value: "-s"},
		},
	})
	assert.Error(t, err)
}

func TestManifest_BuildManifestLDFlags_UnmarshalYAML_Ugly(t *testing.T) {
	var flags buildManifestLDFlags

	assert.NoError(t, flags.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: ""}))
	assert.Nil(t, flags)
}

func TestManifest_LoadManifest_View_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	pub, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	signedView := &ViewManifest{
		Code:    "photo-browser",
		Name:    "Photo Browser",
		Version: ViewVersion("0.1.0"),
		Permissions: ViewPermissions{
			Clipboard:  true,
			Filesystem: true,
		},
	}
	msg, err := viewManifestBytes(signedView)
	assert.NoError(t, err)
	signedView.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))
	m.Files["/.core/view.yaml"] = "code: photo-browser\nname: Photo Browser\nversion: 0.1.0\npermissions:\n  clipboard: true\n  filesystem: true\nsign: " + signedView.Sign + "\n"

	var got ViewManifest
	err = LoadManifest(m, "/.core/view.yaml", &got)
	assert.NoError(t, err)
	assert.Equal(t, "photo-browser", got.Code)
	assert.True(t, got.Permissions.Clipboard)
	assert.True(t, got.Permissions.Filesystem)
}

func TestManifest_LoadManifest_View_VersionInteger_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/view.yaml"] = "code: photo-browser\nname: Photo Browser\nversion: 1\nsign: " + base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)) + "\n"

	var view ViewManifest
	err := LoadManifest(m, "/.core/view.yaml", &view)
	assert.NoError(t, err)
	assert.Equal(t, ViewVersion("1"), view.Version)
}

func TestManifest_LoadManifest_View_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/view.yaml"] = "code: photo-browser\nname: Photo Browser\npermissions:\n  clipboard: true\n"

	var view ViewManifest
	err := LoadManifest(m, "/.core/view.yaml", &view)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsigned view manifest rejected")
}

func TestManifest_LoadManifest_View_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/view.yaml"] = "code: photo-browser\nname: Photo Browser\nsign: " + base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize-1)) + "\npermissions:\n  clipboard: true\n"

	var view ViewManifest
	err := LoadManifest(m, "/.core/view.yaml", &view)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "view manifest signature is not ed25519-sized")
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

func TestManifest_LoadManifest_Package_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\nlicence: EUPL-1.2\nsign: " + base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)) + "\nsign_key: not-hex\n"

	var pkg PackageManifest
	err := LoadManifest(m, "/.core/manifest.yaml", &pkg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode package sign_key failed")
}

func TestManifest_LoadManifest_Package_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")
	pub1, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	_, priv2, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	pkg := &PackageManifest{
		Code:    "go-io",
		Name:    "Core I/O",
		Version: "0.3.0",
		Licence: "EUPL-1.2",
		SignKey: hex.EncodeToString(pub1),
	}
	msg, err := packageManifestBytes(pkg)
	assert.NoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv2, msg))

	out, err := yaml.Marshal(pkg)
	assert.NoError(t, err)
	m.Files["/.core/manifest.yaml"] = string(out)

	var got PackageManifest
	err = LoadManifest(m, "/.core/manifest.yaml", &got)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package manifest signature mismatch")
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

func TestManifest_LoadManifest_PackageSignature_Good(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        "Core I/O",
		Version:     "0.3.0",
		Description: "Mandatory I/O abstraction layer",
		Licence:     "EUPL-1.2",
		SignKey:     hex.EncodeToString(pub),
	}

	msg, err := packageManifestBytes(pkg)
	assert.NoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	m := coreio.NewMockMedium()
	m.Files["/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\ndescription: Mandatory I/O abstraction layer\nlicence: EUPL-1.2\nsign_key: " + pkg.SignKey + "\nsign: " + pkg.Sign + "\n"

	var round PackageManifest
	err = LoadManifest(m, "/.core/manifest.yaml", &round)
	assert.NoError(t, err)
	assert.Equal(t, pkg.Code, round.Code)
	assert.Equal(t, pkg.SignKey, round.SignKey)
}

func TestManifest_LoadManifest_PackageSignature_UntrustedKey_Bad(t *testing.T) {
	trustedPub, _, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	untrustedPub, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(trustedPub))

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        "Core I/O",
		Version:     "0.3.0",
		Description: "Mandatory I/O abstraction layer",
		Licence:     "EUPL-1.2",
		SignKey:     hex.EncodeToString(untrustedPub),
	}

	msg, err := packageManifestBytes(pkg)
	assert.NoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	m := coreio.NewMockMedium()
	m.Files["/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\ndescription: Mandatory I/O abstraction layer\nlicence: EUPL-1.2\nsign_key: " + pkg.SignKey + "\nsign: " + pkg.Sign + "\n"

	var round PackageManifest
	err = LoadManifest(m, "/.core/manifest.yaml", &round)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "package sign_key is not trusted")
}

func TestManifest_LoadManifest_PackageSignature_Bad(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        "Core I/O",
		Version:     "0.3.0",
		Description: "Mandatory I/O abstraction layer",
		Licence:     "EUPL-1.2",
		SignKey:     hex.EncodeToString(pub),
	}

	msg, err := packageManifestBytes(pkg)
	assert.NoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	m := coreio.NewMockMedium()
	m.Files["/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\ndescription: Mandatory I/O abstraction layer\nlicence: EUPL-1.2\nsign_key: " + pkg.SignKey + "\nsign: " + pkg.Sign + "\n"

	// Tamper with the persisted content after signing.
	m.Files["/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\ndescription: Tampered description\nlicence: EUPL-1.2\nsign_key: " + pkg.SignKey + "\nsign: " + pkg.Sign + "\n"

	var round PackageManifest
	err = LoadManifest(m, "/.core/manifest.yaml", &round)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature mismatch")
}

func TestManifest_LoadManifest_ViewSignatureShape_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/view.yaml"] = "code: photo-browser\nname: Photo Browser\nsign: not-base64!!\n"

	var view ViewManifest
	err := LoadManifest(m, "/.core/view.yaml", &view)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid view manifest signature")
}

func TestManifest_LoadManifest_ViewUnsigned_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/view.yaml"] = "code: photo-browser\nname: Photo Browser\n"

	var view ViewManifest
	err := LoadManifest(m, "/.core/view.yaml", &view)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsigned view manifest rejected")
}

func TestManifest_LoadManifest_PackageUnsigned_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\nlicence: EUPL-1.2\n"

	var pkg PackageManifest
	err := LoadManifest(m, "/.core/manifest.yaml", &pkg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsigned package manifest rejected")
}

func TestManifest_LoadManifest_PackageMissingSignKey_Bad(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        "Core I/O",
		Version:     "0.3.0",
		Description: "Mandatory I/O abstraction layer",
		Licence:     "EUPL-1.2",
		SignKey:     hex.EncodeToString(pub),
	}
	msg, err := packageManifestBytes(pkg)
	assert.NoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	m := coreio.NewMockMedium()
	m.Files["/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\ndescription: Mandatory I/O abstraction layer\nlicence: EUPL-1.2\nsign: " + pkg.Sign + "\n"

	var round PackageManifest
	err = LoadManifest(m, "/.core/manifest.yaml", &round)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing package sign_key")
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
