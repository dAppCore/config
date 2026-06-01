package config

import (
	"crypto/ed25519"
	core "dappco.re/go"
	"encoding/base64"
	"encoding/hex"
	"runtime"

	coreio "dappco.re/go/io"
	"gopkg.in/yaml.v3"
)

const (
	manifestTestNotHex                     = "not-hex"
	manifestTestTrustedPubFile             = "trusted.pub"
	manifestTestPhotoBrowserCode           = "photo-browser"
	manifestTestPhotoBrowserName           = "Photo Browser"
	manifestTestCoreIOName                 = "Core I/O"
	manifestTestMandatoryIODescription     = "Mandatory I/O abstraction layer"
	manifestTestEUPL                       = "EUPL-1.2"
	manifestTestDecodePackageSignKeyFailed = "decode package sign_key failed"
	manifestTestSignPrefix                 = "\nsign: "
	manifestTestBuildPath                  = "/.core/" + FileBuild
	manifestTestViewPath                   = "/.core/" + FileView
	manifestTestManifestPath               = "/.core/" + FileManifest
	manifestTestPackageContentPrefix       = "code: go-io\nname: Core I/O\nversion: 0.3.0\ndescription: Mandatory I/O abstraction layer\nlicence: EUPL-1.2\nsign_key: "
	manifestTestKeepMe                     = "keep-me"
)

func setManifestTrustKeys(t *core.T, keys ...string) {
	t.Helper()
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", core.Join(",", keys...))
}

func TestManifest_splitManifestTrustedKeys_Good(t *core.T) {
	got := splitManifestTrustedKeys("a,b;c d\te\nf")
	want := []string{"a", "b", "c", "d", "e", "f"}
	core.AssertEqual(t, want, got)
}

func TestManifest_splitManifestTrustedKeys_Bad(t *core.T) {
	got := splitManifestTrustedKeys("")
	core.AssertEmpty(t, got)
	core.AssertLen(t, got, 0)
}

func TestManifest_splitManifestTrustedKeys_Ugly(t *core.T) {
	got := splitManifestTrustedKeys("   ")
	core.AssertEmpty(t, got)
	core.AssertLen(t, got, 0)
}

func TestManifest_parseManifestPublicKey_Good(t *core.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)

	got, err := publicKeyResult(parseManifestPublicKey(hex.EncodeToString(pub)))
	core.AssertNoError(t, err)
	core.AssertEqual(t, hex.EncodeToString(pub), hex.EncodeToString(got))
}

func TestManifest_parseManifestPublicKey_Bad(t *core.T) {
	_, err := publicKeyResult(parseManifestPublicKey(manifestTestNotHex))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "decode manifest public key failed")
}

func TestManifest_parseManifestPublicKey_Ugly(t *core.T) {
	_, err := publicKeyResult(parseManifestPublicKey("   "))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "empty manifest public key")
}

func TestManifest_dedupeManifestKeys_Good(t *core.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	got := dedupeManifestKeys([]ed25519.PublicKey{pub, pub})
	core.AssertEqual(t, []ed25519.PublicKey{pub}, got)
}

func TestManifest_dedupeManifestKeys_Bad(t *core.T) {
	out := dedupeManifestKeys(nil)
	core.AssertEmpty(t, out)
	core.AssertLen(t, out, 0)
}

func TestManifest_dedupeManifestKeys_Ugly(t *core.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)

	invalid := ed25519.PublicKey("short")
	out := dedupeManifestKeys([]ed25519.PublicKey{invalid, invalid, pub})
	core.AssertEqual(t, []ed25519.PublicKey{pub}, out)
}

func TestManifest_missingOrEmptyStringField_Good(t *core.T) {
	raw := map[string]any{"sign": "abc"}
	got := missingOrEmptyStringField(raw, "sign", "abc")
	core.AssertFalse(t, got)
}

func TestManifest_missingOrEmptyStringField_Bad(t *core.T) {
	raw := map[string]any{}
	got := missingOrEmptyStringField(raw, "sign", "abc")
	core.AssertTrue(t, got)
}

func TestManifest_missingOrEmptyStringField_Ugly(t *core.T) {
	raw := map[string]any{"sign": ""}
	core.AssertTrue(t, missingOrEmptyStringField(raw, "sign", "abc"))
	raw["sign"] = "   "
	core.AssertTrue(t, missingOrEmptyStringField(raw, "sign", "abc"))
}

func setManifestHomeDir(t *core.T, home string) {
	t.Helper()
	previous := manifestHomeDir
	manifestHomeDir = func() string {
		return home
	}
	t.Cleanup(func() {
		manifestHomeDir = previous
	})
}

func TestManifest_TrustedManifestPublicKeys_Good(t *core.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	got, err := trustedKeysResult(TrustedManifestPublicKeys())
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 1)
	core.AssertEqual(t, pub, got[0])
}

func TestManifest_TrustedManifestPublicKeys_Bad(t *core.T) {
	setManifestTrustKeys(t, manifestTestNotHex)
	_, err := trustedKeysResult(TrustedManifestPublicKeys())
	core.AssertError(t, err)
}

func TestManifest_TrustedManifestPublicKeys_Ugly(t *core.T) {
	home := t.TempDir()
	setManifestHomeDir(t, home)
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")

	keysDir := core.PathJoin(home, ".core", "keys")
	testMkdirAll(t, keysDir, 0o755)

	pub, _, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	testWriteFile(t, core.PathJoin(keysDir, manifestTestTrustedPubFile), []byte(core.Sprintf("%x\n", pub)), 0o644)

	got, err := trustedKeysResult(TrustedManifestPublicKeys())
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 1)
}

func TestManifest_TrustedManifestPublicKeys_SymlinkedCore_Bad(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip(serviceTestWindowsSymlinkSkipMessage)
	}

	home := t.TempDir()
	setManifestHomeDir(t, home)
	coreDir := core.PathJoin(home, ".core")

	realCore := core.PathJoin(t.TempDir(), "real-core")
	testMkdirAll(t, realCore, 0o755)
	testSymlink(t, realCore, coreDir)
	t.Cleanup(func() { testRemove(coreDir) })

	_, err := trustedKeysResult(TrustedManifestPublicKeys())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "symlinked .core directory rejected")
}

func TestManifest_TrustedManifestPublicKeys_SymlinkedKeysDir_Bad(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip(serviceTestWindowsSymlinkSkipMessage)
	}

	home := t.TempDir()
	setManifestHomeDir(t, home)
	realKeys := core.PathJoin(t.TempDir(), "real-keys")

	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")

	coreDir := core.PathJoin(home, ".core")
	keysDir := core.PathJoin(coreDir, "keys")
	testMkdirAll(t, coreDir, 0o755)
	testMkdirAll(t, realKeys, 0o755)
	testSymlink(t, realKeys, keysDir)
	t.Cleanup(func() { testRemove(keysDir) })

	_, err := trustedKeysResult(TrustedManifestPublicKeys())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "symlinked trusted keys directory rejected")
}

func TestManifest_TrustedManifestPublicKeys_SymlinkedKeyFile_Bad(t *core.T) {
	if runtime.GOOS == "windows" {
		t.Skip(serviceTestWindowsSymlinkSkipMessage)
	}

	home := t.TempDir()
	setManifestHomeDir(t, home)
	realKeys := core.PathJoin(t.TempDir(), "real-keys")
	pub, _, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)

	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")

	coreDir := core.PathJoin(home, ".core")
	keysDir := core.PathJoin(coreDir, "keys")
	testMkdirAll(t, keysDir, 0o755)
	testMkdirAll(t, realKeys, 0o755)
	testWriteFile(t, core.PathJoin(realKeys, manifestTestTrustedPubFile), []byte(core.Sprintf("%x\n", pub)), 0o644)
	symlinkPath := core.PathJoin(keysDir, manifestTestTrustedPubFile)
	testSymlink(t, core.PathJoin(realKeys, manifestTestTrustedPubFile), symlinkPath)
	t.Cleanup(func() { testRemove(symlinkPath) })

	_, err = trustedKeysResult(TrustedManifestPublicKeys())
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "symlinked trusted key rejected")
}

func TestManifest_TrustedManifestPublicKeys_Exported_Good(t *core.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	got, err := trustedKeysResult(TrustedManifestPublicKeys())
	core.AssertNoError(t, err)
	core.AssertLen(t, got, 1)
	core.AssertEqual(t, pub, got[0])
}

func TestManifest_SignViewManifest_ViewSignatureHelpers_Good(t *core.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)

	view := &ViewManifest{
		Code:    manifestTestPhotoBrowserCode,
		Name:    manifestTestPhotoBrowserName,
		Version: ViewVersion("0.1.0"),
		Layout:  "HLCRF",
		Slots: map[string]any{
			"C": "photo-grid",
		},
	}

	body, err := bytesResult(CanonicalViewManifestBytes(view))
	core.AssertNoError(t, err)
	core.AssertContains(t, string(body), "sign: \"\"")

	err = resultError(SignViewManifest(view, priv))
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, view.Sign)
	core.AssertNoError(t, resultError(ValidateViewManifestSignature(view)))
	core.AssertNoError(t, resultError(VerifyViewManifestSignature(view, pub)))
}

func TestManifest_ValidateViewManifestSignature_ViewSignatureHelpers_Bad(t *core.T) {
	view := &ViewManifest{
		Code: manifestTestPhotoBrowserCode,
		Name: manifestTestPhotoBrowserName,
		Sign: "not-base64!!",
	}

	err := resultError(ValidateViewManifestSignature(view))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid view manifest signature")
}

func TestManifest_VerifyViewManifestSignature_ViewSignatureHelpers_Ugly(t *core.T) {
	pub, _, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)

	view := &ViewManifest{
		Code: manifestTestPhotoBrowserCode,
		Name: manifestTestPhotoBrowserName,
		Sign: base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
	}

	err = resultError(VerifyViewManifestSignature(view, pub[:ed25519.PublicKeySize-1]))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not an ed25519 public key")
}

func TestManifest_SignPackageManifest_PackageSignatureHelpers_Good(t *core.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        manifestTestCoreIOName,
		Version:     "0.3.0",
		Description: manifestTestMandatoryIODescription,
		Licence:     manifestTestEUPL,
	}

	body, err := bytesResult(CanonicalPackageManifestBytes(pkg))
	core.AssertNoError(t, err)
	core.AssertContains(t, string(body), "sign: \"\"")
	core.AssertContains(t, string(body), "sign_key: \"\"")

	err = resultError(SignPackageManifest(pkg, priv))
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, pkg.Sign)
	core.AssertNotEmpty(t, pkg.SignKey)
	core.AssertNoError(t, resultError(VerifyPackageManifest(pkg)))
}

func TestManifest_VerifyPackageManifest_PackageSignatureHelpers_Bad(t *core.T) {
	pkg := &PackageManifest{
		Code:    "go-io",
		Name:    manifestTestCoreIOName,
		Version: "0.3.0",
		SignKey: manifestTestNotHex,
		Sign:    base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)),
	}

	err := resultError(VerifyPackageManifest(pkg))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), manifestTestDecodePackageSignKeyFailed)
}

func TestManifest_VerifyPackageManifest_PackageSignatureHelpers_Ugly(t *core.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        manifestTestCoreIOName,
		Version:     "0.3.0",
		Description: manifestTestMandatoryIODescription,
		Licence:     manifestTestEUPL,
	}

	err = resultError(SignPackageManifest(pkg, priv))
	core.AssertNoError(t, err)
	pkg.Description = "Tampered"

	err = resultError(VerifyPackageManifest(pkg))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "signature mismatch")
}

func TestManifest_LoadManifest_Good(t *core.T) {
	m := coreio.NewMockMedium()
	pub, priv, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	signedPkg := &PackageManifest{
		Code:    "go-io",
		Name:    manifestTestCoreIOName,
		Version: "0.3.0",
		Licence: manifestTestEUPL,
		SignKey: hex.EncodeToString(pub),
	}
	msg, err := bytesResult(packageManifestBytes(signedPkg))
	core.AssertNoError(t, err)
	signedPkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))
	m.Files["/pkg/.core/manifest.yaml"] = "code: go-io\nname: Core I/O\nversion: 0.3.0\nlicence: EUPL-1.2\nsign_key: " + signedPkg.SignKey + manifestTestSignPrefix + signedPkg.Sign + "\n"

	var pkg PackageManifest
	err = resultError(LoadManifest(m, "/pkg/.core/manifest.yaml", &pkg))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "go-io", pkg.Code)
	core.AssertEqual(t, manifestTestCoreIOName, pkg.Name)
	core.AssertEqual(t, "0.3.0", pkg.Version)
	core.AssertEqual(t, manifestTestEUPL, pkg.Licence)
}

func TestManifest_LoadManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	var pkg PackageManifest
	err := resultError(LoadManifest(m, "/nonexistent.yaml", &pkg))
	core.AssertError(t, err)
}

func TestManifest_LoadManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/bad.yaml"] = "this is: [not: valid: yaml"

	var pkg PackageManifest
	err := resultError(LoadManifest(m, "/bad.yaml", &pkg))
	core.AssertError(t, err)
}

func TestManifest_LoadManifest_Build_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestBuildPath] = "name: core\noutput: dist\ncgo: false\ntargets:\n  - os: linux\n    arch: amd64\n  - os: darwin\n    arch: arm64\n"

	var build BuildManifest
	err := resultError(LoadManifest(m, manifestTestBuildPath, &build))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", build.Name)
	core.AssertEqual(t, "dist", build.Output)
	core.AssertLen(t, build.Targets, 2)
	core.AssertEqual(t, "linux", build.Targets[0].OS)
	core.AssertEqual(t, "amd64", build.Targets[0].Arch)
}

func TestManifest_LoadManifest_Build_ShorthandTargets_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestBuildPath] = "name: core\noutput: dist\ntargets:\n  - linux/amd64\n  - darwin/arm64\nsign:\n  enabled: true\n  gpg:\n    key: $GPG_KEY_ID\n  macos:\n    identity: 'Developer ID Application: Example'\n    notarize: false\nsdk:\n  spec: openapi.yaml\n  languages:\n    - typescript\n    - go\n  output: sdk/\n  diff: true\n"

	var build BuildManifest
	err := resultError(LoadManifest(m, manifestTestBuildPath, &build))
	core.AssertNoError(t, err)
	core.AssertLen(t, build.Targets, 2)
	core.AssertEqual(t, "linux", build.Targets[0].OS)
	core.AssertEqual(t, "amd64", build.Targets[0].Arch)
	core.AssertTrue(t, build.Signing.Enabled)
	core.AssertEqual(t, "$GPG_KEY_ID", build.Signing.GPG.Key)
	core.AssertEqual(t, "Developer ID Application: Example", build.Signing.MacOS.Identity)
	core.AssertTrue(t, build.SDK.Diff)
	core.AssertEqual(t, "openapi.yaml", build.SDK.Spec)
	core.AssertEqual(t, []string{"typescript", "go"}, build.SDK.Languages)
}

func TestManifest_LoadManifest_Build_LegacyFlat_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestBuildPath] = "name: core\nmain: ./cmd/core\nbinary: core\noutput: dist\nflags:\n  - -trimpath\nldflags: -s -w\ncgo: false\ntargets:\n  - linux/amd64\n"

	var build BuildManifest
	err := resultError(LoadManifest(m, manifestTestBuildPath, &build))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", build.Name)
	core.AssertEqual(t, "./cmd/core", build.Main)
	core.AssertEqual(t, "core", build.Binary)
	core.AssertEqual(t, "dist", build.Output)
	core.AssertEqual(t, []string{"-trimpath"}, build.Flags)
	core.AssertEqual(t, "-s -w", build.LDFlags)
	core.AssertFalse(t, build.CGO)
	core.AssertLen(t, build.Targets, 1)
}

func TestManifest_BuildTarget_UnmarshalYAML_Good(t *core.T) {
	var target BuildTarget
	core.AssertNoError(t, resultError(target.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "linux/amd64"})))
	core.AssertEqual(t, BuildTarget{OS: "linux", Arch: "amd64"}, target)

	core.AssertNoError(t, resultError(target.UnmarshalYAML(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "o" + "s"},
			{Kind: yaml.ScalarNode, Value: "darwin"},
			{Kind: yaml.ScalarNode, Value: "arch"},
			{Kind: yaml.ScalarNode, Value: "arm64"},
		},
	})))
	core.AssertEqual(t, BuildTarget{OS: "darwin", Arch: "arm64"}, target)
}

func TestManifest_BuildTarget_UnmarshalYAML_Bad(t *core.T) {
	var target BuildTarget

	err := resultError(target.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "linux"}))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid target shorthand")
}

func TestManifest_BuildTarget_UnmarshalYAML_Ugly(t *core.T) {
	var target BuildTarget

	core.AssertNoError(t, resultError(target.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: ""})))
	core.AssertEqual(t, BuildTarget{}, target)
}

// TestManifest_manifestYAMLRoot_Good asserts a document node unwraps to its
// first content node and a plain node is returned as-is.
func TestManifest_manifestYAMLRoot_Good(t *core.T) {
	inner := &yaml.Node{Kind: yaml.ScalarNode, Value: "leaf"}
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{inner}}
	core.AssertSame(t, inner, manifestYAMLRoot(doc))
	core.AssertSame(t, inner, manifestYAMLRoot(inner))
}

// TestManifest_manifestYAMLRoot_Alias asserts an alias node unwraps to its
// anchored target.
func TestManifest_manifestYAMLRoot_Alias(t *core.T) {
	target := &yaml.Node{Kind: yaml.MappingNode}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: target}
	core.AssertSame(t, target, manifestYAMLRoot(alias))
}

// TestManifest_manifestYAMLRoot_Ugly asserts a nil node yields a fresh empty
// node rather than panicking.
func TestManifest_manifestYAMLRoot_Ugly(t *core.T) {
	got := manifestYAMLRoot(nil)
	core.AssertNotNil(t, got)
	core.AssertEqual(t, yaml.Kind(0), got.Kind)
}

func TestManifest_BuildManifestLDFlags_String_Good(t *core.T) {
	flags := buildmanifestldflags{"-s", "-w"}
	got := flags.String()
	core.AssertEqual(t, "-s -w", got)
}

func TestManifest_BuildManifestLDFlags_UnmarshalYAML_Good(t *core.T) {
	var flags buildmanifestldflags

	core.AssertNoError(t, resultError(flags.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "-s -w"})))
	core.AssertEqual(t, buildmanifestldflags{"-s -w"}, flags)

	core.AssertNoError(t, resultError(flags.UnmarshalYAML(&yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "-s"},
			{Kind: yaml.ScalarNode, Value: "-w"},
		},
	})))
	core.AssertEqual(t, buildmanifestldflags{"-s", "-w"}, flags)
}

func TestManifest_BuildManifestLDFlags_UnmarshalYAML_Bad(t *core.T) {
	var flags buildmanifestldflags

	err := resultError(flags.UnmarshalYAML(&yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "0"},
			{Kind: yaml.ScalarNode, Value: "-s"},
		},
	}))
	core.AssertError(t, err)
}

func TestManifest_BuildManifestLDFlags_UnmarshalYAML_Ugly(t *core.T) {
	var flags buildmanifestldflags

	core.AssertNoError(t, resultError(flags.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: ""})))
	core.AssertNil(t, flags)
}

func TestManifest_LoadManifest_View_Good(t *core.T) {
	m := coreio.NewMockMedium()
	pub, priv, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	signedView := &ViewManifest{
		Code:    manifestTestPhotoBrowserCode,
		Name:    manifestTestPhotoBrowserName,
		Version: ViewVersion("0.1.0"),
		Permissions: ViewPermissions{
			Clipboard:  true,
			Filesystem: true,
		},
	}
	msg, err := bytesResult(viewManifestBytes(signedView))
	core.AssertNoError(t, err)
	signedView.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))
	m.Files[manifestTestViewPath] = "code: photo-browser\nname: Photo Browser\nversion: 0.1.0\npermissions:\n  clipboard: true\n  filesystem: true\nsign: " + signedView.Sign + "\n"

	var got ViewManifest
	err = resultError(LoadManifest(m, manifestTestViewPath, &got))
	core.AssertNoError(t, err)
	core.AssertEqual(t, manifestTestPhotoBrowserCode, got.Code)
	core.AssertTrue(t, got.Permissions.Clipboard)
	core.AssertTrue(t, got.Permissions.Filesystem)
}

func TestManifest_LoadManifest_View_VersionInteger_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestViewPath] = "code: photo-browser\nname: Photo Browser\nversion: 1\nsign: " + base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)) + "\n"

	var view ViewManifest
	err := resultError(LoadManifest(m, manifestTestViewPath, &view))
	core.AssertNoError(t, err)
	core.AssertEqual(t, ViewVersion("1"), view.Version)
}

func TestManifest_LoadManifest_View_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestViewPath] = "code: photo-browser\nname: Photo Browser\npermissions:\n  clipboard: true\n"

	var view ViewManifest
	err := resultError(LoadManifest(m, manifestTestViewPath, &view))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsigned view manifest rejected")
}

func TestManifest_LoadManifest_View_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestViewPath] = "code: photo-browser\nname: Photo Browser\nsign: " + base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize-1)) + "\npermissions:\n  clipboard: true\n"

	var view ViewManifest
	err := resultError(LoadManifest(m, manifestTestViewPath, &view))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "view manifest signature is not ed25519-sized")
}

func TestManifest_LoadManifest_Test_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/test.yaml"] = "version: 1\ncommands:\n  - name: unit\n    run: vendor/bin/pest --parallel\n  - name: types\n    run: vendor/bin/phpstan analyse\nenv:\n  APP_ENV: testing\n  DB_CONNECTION: sqlite\n"

	var test TestManifest
	err := resultError(LoadManifest(m, "/.core/test.yaml", &test))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, test.Version)
	core.AssertLen(t, test.Commands, 2)
	core.AssertEqual(t, "unit", test.Commands[0].Name)
	core.AssertEqual(t, "vendor/bin/pest --parallel", test.Commands[0].Run)
	core.AssertEqual(t, "testing", test.Env["APP_ENV"])
}

func TestManifest_LoadManifest_Run_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/run.yaml"] = "version: 1\nservices:\n  - name: database\n    image: postgres:16\n    port: 5432\n    env:\n      POSTGRES_DB: core_dev\ndev:\n  command: php artisan serve\n  port: 8000\n  watch:\n    - app/\n    - resources/\nenv:\n  APP_ENV: local\n"

	var run RunManifest
	err := resultError(LoadManifest(m, "/.core/run.yaml", &run))
	core.AssertNoError(t, err)
	core.AssertEqual(t, 1, run.Version)
	core.AssertLen(t, run.Services, 1)
	core.AssertEqual(t, "database", run.Services[0].Name)
	core.AssertEqual(t, 5432, run.Services[0].Port)
	core.AssertEqual(t, "core_dev", run.Services[0].Env["POSTGRES_DB"])
	core.AssertEqual(t, "php artisan serve", run.Dev.Command)
	core.AssertEqual(t, 8000, run.Dev.Port)
	core.AssertContains(t, run.Dev.Watch, "app/")
}

func TestManifest_LoadManifest_Repos_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/Code/.core/repos.yaml"] = "org: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n    branch: dev\n    type: lib\n    depends:\n      - go-io\n  - path: core/config\n    remote: ssh://forge.example/core/config.git\n    branch: dev\n"

	var repos ReposManifest
	err := resultError(LoadManifest(m, "/Code/.core/repos.yaml", &repos))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "host-uk", repos.Org)
	core.AssertLen(t, repos.Repos, 2)
	core.AssertEqual(t, "core/go", repos.Repos[0].Path)
	core.AssertEqual(t, "dev", repos.Repos[0].Branch)
	core.AssertEqual(t, "lib", repos.Repos[0].Type)
	core.AssertContains(t, repos.Repos[0].Depends, "go-io")
}

func TestManifest_LoadManifest_Repos_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	var repos ReposManifest
	err := resultError(LoadManifest(m, "/missing/repos.yaml", &repos))
	core.AssertError(t, err)
}

func TestManifest_LoadManifest_Package_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestManifestPath] = "code: go-io\nname: Core I/O\nversion: 0.3.0\nlicence: EUPL-1.2\nsign: " + base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)) + "\nsign_key: not-hex\n"

	var pkg PackageManifest
	err := resultError(LoadManifest(m, manifestTestManifestPath, &pkg))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), manifestTestDecodePackageSignKeyFailed)
}

func TestManifest_LoadManifest_Package_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	t.Setenv("CORE_MANIFEST_TRUST_KEYS", "")
	pub1, _, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	_, priv2, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)

	pkg := &PackageManifest{
		Code:    "go-io",
		Name:    manifestTestCoreIOName,
		Version: "0.3.0",
		Licence: manifestTestEUPL,
		SignKey: hex.EncodeToString(pub1),
	}
	msg, err := bytesResult(packageManifestBytes(pkg))
	core.AssertNoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv2, msg))

	out, err := yaml.Marshal(pkg)
	core.AssertNoError(t, err)
	m.Files[manifestTestManifestPath] = string(out)

	var got PackageManifest
	err = resultError(LoadManifest(m, manifestTestManifestPath, &got))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "package manifest signature mismatch")
}

func TestManifest_LoadManifest_Release_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/.core/release.yaml"] = "archive:\n  format: tar.gz\n  include:\n    - LICENSE.txt\n    - README.md\nchecksums: true\ngithub:\n  draft: false\n  prerelease: false\nchangelog:\n  include:\n    - feat\n    - fix\n"

	var rel ReleaseManifest
	err := resultError(LoadManifest(m, "/.core/release.yaml", &rel))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "tar.gz", rel.Archive.Format)
	core.AssertContains(t, rel.Archive.Include, "LICENSE.txt")
	core.AssertTrue(t, rel.Checksums)
	core.AssertFalse(t, rel.GitHub.Draft)
	core.AssertContains(t, rel.Changelog.Include, "feat")
}

func TestManifest_LoadManifest_Agent_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/home/.core/agent.yaml"] = "daemon:\n  enabled: true\n  watch:\n    - ~/Code/core/\n  schedule:\n    - cron: '*/5 * * * *'\n      action: health.check\n  mcp:\n    port: 8080\n  api:\n    port: 8099\n    bind: 127.0.0.1\nagents:\n  codex:\n    total: 2\n  claude:\n    total: 1\n"

	var agent AgentManifest
	err := resultError(LoadManifest(m, "/home/.core/agent.yaml", &agent))
	core.AssertNoError(t, err)
	core.AssertTrue(t, agent.Daemon.Enabled)
	core.AssertEqual(t, "health.check", agent.Daemon.Schedule[0].Action)
	core.AssertEqual(t, 8099, agent.Daemon.API.Port)
	core.AssertEqual(t, 2, agent.Agents["codex"].Total)
}

func TestManifest_LoadManifest_Zone_Good(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files["/home/.core/zone.yaml"] = "zone:\n  name: snider\n  identity: '@snider@lthn'\n  chain:\n    mode: thin\n    daemon: localhost:36941\n  network:\n    wireguard:\n      interface: wg-lthn\n      listen: 51820\n  services:\n    vpn:\n      enabled: true\n      price: 0.001\n      capacity: 100\n    dns:\n      enabled: true\n    compute:\n      enabled: true\n      models:\n        - lem-1b\n        - lem-4b\n  staking:\n    amount: 1000\n    tier: trusted\n"

	var zone ZoneManifest
	err := resultError(LoadManifest(m, "/home/.core/zone.yaml", &zone))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "snider", zone.Zone.Name)
	core.AssertEqual(t, "thin", zone.Zone.Chain.Mode)
	core.AssertEqual(t, "wg-lthn", zone.Zone.Network.WireGuard.Interface)
	core.AssertEqual(t, 100, zone.Zone.Services.VPN.Capacity)
	core.AssertContains(t, zone.Zone.Services.Compute.Models, "lem-4b")
}

func TestManifest_LoadManifest_Schema_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestBuildPath] = "targets: 42\n"

	var build BuildManifest
	err := resultError(LoadManifest(m, manifestTestBuildPath, &build))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "schema validation failed")
}

func TestManifest_LoadManifest_PackageSignature_Good(t *core.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        manifestTestCoreIOName,
		Version:     "0.3.0",
		Description: manifestTestMandatoryIODescription,
		Licence:     manifestTestEUPL,
		SignKey:     hex.EncodeToString(pub),
	}

	msg, err := bytesResult(packageManifestBytes(pkg))
	core.AssertNoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	m := coreio.NewMockMedium()
	m.Files[manifestTestManifestPath] = manifestTestPackageContentPrefix + pkg.SignKey + manifestTestSignPrefix + pkg.Sign + "\n"

	var round PackageManifest
	err = resultError(LoadManifest(m, manifestTestManifestPath, &round))
	core.AssertNoError(t, err)
	core.AssertEqual(t, pkg.Code, round.Code)
	core.AssertEqual(t, pkg.SignKey, round.SignKey)
}

func TestManifest_LoadManifest_PackageSignature_UntrustedKey_Bad(t *core.T) {
	trustedPub, _, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	untrustedPub, priv, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(trustedPub))

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        manifestTestCoreIOName,
		Version:     "0.3.0",
		Description: manifestTestMandatoryIODescription,
		Licence:     manifestTestEUPL,
		SignKey:     hex.EncodeToString(untrustedPub),
	}

	msg, err := bytesResult(packageManifestBytes(pkg))
	core.AssertNoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	m := coreio.NewMockMedium()
	m.Files[manifestTestManifestPath] = manifestTestPackageContentPrefix + pkg.SignKey + manifestTestSignPrefix + pkg.Sign + "\n"

	var round PackageManifest
	err = resultError(LoadManifest(m, manifestTestManifestPath, &round))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "package sign_key is not trusted")
}

func TestManifest_LoadManifest_PackageSignature_Bad(t *core.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        manifestTestCoreIOName,
		Version:     "0.3.0",
		Description: manifestTestMandatoryIODescription,
		Licence:     manifestTestEUPL,
		SignKey:     hex.EncodeToString(pub),
	}

	msg, err := bytesResult(packageManifestBytes(pkg))
	core.AssertNoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	m := coreio.NewMockMedium()
	m.Files[manifestTestManifestPath] = manifestTestPackageContentPrefix + pkg.SignKey + manifestTestSignPrefix + pkg.Sign + "\n"

	// Tamper with the persisted content after signing.
	m.Files[manifestTestManifestPath] = "code: go-io\nname: Core I/O\nversion: 0.3.0\ndescription: Tampered description\nlicence: EUPL-1.2\nsign_key: " + pkg.SignKey + manifestTestSignPrefix + pkg.Sign + "\n"

	var round PackageManifest
	err = resultError(LoadManifest(m, manifestTestManifestPath, &round))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "signature mismatch")
}

func TestManifest_LoadManifest_ViewSignatureShape_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestViewPath] = "code: photo-browser\nname: Photo Browser\nsign: not-base64!!\n"

	var view ViewManifest
	err := resultError(LoadManifest(m, manifestTestViewPath, &view))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid view manifest signature")
}

func TestManifest_LoadManifest_ViewUnsigned_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestViewPath] = "code: photo-browser\nname: Photo Browser\n"

	var view ViewManifest
	err := resultError(LoadManifest(m, manifestTestViewPath, &view))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsigned view manifest rejected")
}

func TestManifest_LoadManifest_PackageUnsigned_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	m.Files[manifestTestManifestPath] = "code: go-io\nname: Core I/O\nversion: 0.3.0\nlicence: EUPL-1.2\n"

	var pkg PackageManifest
	err := resultError(LoadManifest(m, manifestTestManifestPath, &pkg))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsigned package manifest rejected")
}

func TestManifest_LoadManifest_PackageMissingSignKey_Bad(t *core.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	core.AssertNoError(t, err)
	setManifestTrustKeys(t, hex.EncodeToString(pub))

	pkg := &PackageManifest{
		Code:        "go-io",
		Name:        manifestTestCoreIOName,
		Version:     "0.3.0",
		Description: manifestTestMandatoryIODescription,
		Licence:     manifestTestEUPL,
		SignKey:     hex.EncodeToString(pub),
	}
	msg, err := bytesResult(packageManifestBytes(pkg))
	core.AssertNoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	m := coreio.NewMockMedium()
	m.Files[manifestTestManifestPath] = "code: go-io\nname: Core I/O\nversion: 0.3.0\ndescription: Mandatory I/O abstraction layer\nlicence: EUPL-1.2\nsign: " + pkg.Sign + "\n"

	var round PackageManifest
	err = resultError(LoadManifest(m, manifestTestManifestPath, &round))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "missing package sign_key")
}

func TestManifest_KnownFiles_Good(t *core.T) {
	// The constants are single-source-of-truth names; KnownFiles must contain
	// every canonical project-level file and not duplicate any.
	core.AssertContains(t, KnownFiles, FileConfig)
	core.AssertContains(t, KnownFiles, FileBuild)
	core.AssertContains(t, KnownFiles, FileTest)
	core.AssertContains(t, KnownFiles, FileRun)
	core.AssertContains(t, KnownFiles, FileRelease)
	core.AssertContains(t, KnownFiles, FileView)
	core.AssertContains(t, KnownFiles, FileManifest)
	core.AssertContains(t, KnownFiles, FileWorkspace)
	core.AssertContains(t, KnownFiles, FileRepos)
	core.AssertContains(t, KnownFiles, FileIDE)
	core.AssertContains(t, KnownFiles, FilePHP)
	core.AssertEqual(t, ".core", Directory)

	// User-level files have constants but are not part of project discovery.
	core.AssertEqual(t, "agent.yaml", FileAgent)
	core.AssertEqual(t, "zone.yaml", FileZone)
	core.AssertEqual(t, "ide.yaml", FileIDE)
	core.AssertEqual(t, "php.yaml", FilePHP)

	seen := map[string]struct{}{}
	for _, name := range KnownFiles {
		_, dup := seen[name]
		core.AssertFalse(t, dup, "duplicate known file: %s", name)
		seen[name] = struct{}{}
	}
}

func axManifestView() ViewManifest {
	return ViewManifest{
		Version:   "1",
		Code:      manifestTestPhotoBrowserCode,
		Name:      manifestTestPhotoBrowserName,
		Title:     "Photos",
		Width:     800,
		Height:    600,
		Resizable: true,
	}
}

func axManifestPackage() PackageManifest {
	return PackageManifest{
		Code:        "go-config",
		Name:        "Core Config",
		Module:      "dappco.re/go/config",
		Version:     "0.9.0",
		Description: "config package",
		Licence:     manifestTestEUPL,
	}
}

func axSignedView(t *core.T) (ViewManifest, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	view := axManifestView()
	core.RequireNoError(t, resultError(SignViewManifest(&view, priv)))
	return view, pub
}

func axSignedPackage(t *core.T) (PackageManifest, ed25519.PublicKey) {
	t.Helper()
	pub, priv, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	pkg := axManifestPackage()
	core.RequireNoError(t, resultError(SignPackageManifest(&pkg, priv)))
	return pkg, pub
}

func testYAMLRoot(t *core.T, body string) *yaml.Node {
	t.Helper()
	var node yaml.Node
	core.RequireNoError(t, yaml.Unmarshal([]byte(body), &node))
	return manifestYAMLRoot(&node)
}

func TestManifest_ViewVersion_UnmarshalYAML_Good(t *core.T) {
	var version ViewVersion
	err := resultError(version.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "1"}))
	core.AssertNoError(t, err)
	core.AssertEqual(t, ViewVersion("1"), version)
}

func TestManifest_ViewVersion_UnmarshalYAML_Bad(t *core.T) {
	var version ViewVersion
	err := resultError(version.UnmarshalYAML(&yaml.Node{Kind: yaml.SequenceNode}))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid view manifest version")
}

func TestManifest_ViewVersion_UnmarshalYAML_Ugly(t *core.T) {
	base := &yaml.Node{Kind: yaml.ScalarNode, Value: "2"}
	alias := &yaml.Node{Kind: yaml.AliasNode, Alias: base}
	var version ViewVersion
	err := resultError(version.UnmarshalYAML(alias))
	core.AssertNoError(t, err)
	core.AssertEqual(t, ViewVersion("2"), version)
}

func TestManifest_buildmanifestldflags_UnmarshalYAML_Good(t *core.T) {
	var flags buildmanifestldflags
	err := resultError(flags.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "-s -w"}))
	core.AssertNoError(t, err)
	core.AssertEqual(t, buildmanifestldflags{"-s -w"}, flags)
}

func TestManifest_buildmanifestldflags_UnmarshalYAML_Bad(t *core.T) {
	var flags buildmanifestldflags
	err := resultError(flags.UnmarshalYAML(&yaml.Node{Kind: yaml.MappingNode}))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsupported ldflags mapping")
}

func TestManifest_buildmanifestldflags_UnmarshalYAML_Ugly(t *core.T) {
	var flags buildmanifestldflags
	err := resultError(flags.UnmarshalYAML(&yaml.Node{
		Kind: yaml.SequenceNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "-s"},
			{Kind: yaml.ScalarNode, Value: "-w"},
		},
	}))
	core.AssertNoError(t, err)
	core.AssertEqual(t, buildmanifestldflags{"-s", "-w"}, flags)
}

func TestManifest_buildmanifestldflags_String_Good(t *core.T) {
	flags := buildmanifestldflags{"-s", "-w"}
	got := flags.String()
	core.AssertEqual(t, "-s -w", got)
	core.AssertNotEmpty(t, got)
}

func TestManifest_buildmanifestldflags_String_Bad(t *core.T) {
	var flags buildmanifestldflags
	got := flags.String()
	core.AssertEqual(t, "", got)
}

func TestManifest_buildmanifestldflags_String_Ugly(t *core.T) {
	flags := buildmanifestldflags{"-X", "main.version=0.9.0"}
	got := flags.String()
	core.AssertEqual(t, "-X main.version=0.9.0", got)
}

func TestManifest_BuildManifest_UnmarshalYAML_Good(t *core.T) {
	var build BuildManifest
	body := "version: 1\nproject:\n  name: core\n  main: ./cmd/core\nbuild:\n  flags: [-trimpath]\n  ldflags: [-s, -w]\ntargets:\n  - linux/amd64\n"
	err := resultError(build.UnmarshalYAML(testYAMLRoot(t, body)))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "core", build.Name)
}

func TestManifest_BuildManifest_UnmarshalYAML_Bad(t *core.T) {
	var build BuildManifest
	body := "version: 1\ntargets:\n  - invalid-target\n"
	err := resultError(build.UnmarshalYAML(testYAMLRoot(t, body)))
	core.AssertError(t, err)
}

func TestManifest_BuildManifest_UnmarshalYAML_Ugly(t *core.T) {
	var build BuildManifest
	body := "version: 1\nname: legacy\nmain: ./main.go\nbinary: app\noutput: dist\nldflags: -s -w\ncgo: true\n"
	err := resultError(build.UnmarshalYAML(testYAMLRoot(t, body)))
	core.AssertNoError(t, err)
	core.AssertEqual(t, "legacy", build.Project.Name)
}

func TestManifest_CanonicalViewManifestBytes_Good(t *core.T) {
	view := axManifestView()
	view.Sign = "signature"
	body, err := bytesResult(CanonicalViewManifestBytes(&view))
	core.AssertNoError(t, err)
	core.AssertNotContains(t, string(body), "signature")
}

func TestManifest_CanonicalViewManifestBytes_Bad(t *core.T) {
	body, err := bytesResult(CanonicalViewManifestBytes(nil))
	core.AssertNoError(t, err)
	core.AssertContains(t, string(body), "null")
}

func TestManifest_CanonicalViewManifestBytes_Ugly(t *core.T) {
	view := axManifestView()
	view.Sign = manifestTestKeepMe
	_, err := bytesResult(CanonicalViewManifestBytes(&view))
	core.AssertNoError(t, err)
	core.AssertEqual(t, manifestTestKeepMe, view.Sign)
}

func TestManifest_ValidateViewManifestSignature_Good(t *core.T) {
	view, _ := axSignedView(t)
	err := resultError(ValidateViewManifestSignature(&view))
	core.AssertNoError(t, err)
}

func TestManifest_ValidateViewManifestSignature_Bad(t *core.T) {
	view := axManifestView()
	err := resultError(ValidateViewManifestSignature(&view))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsigned")
}

func TestManifest_ValidateViewManifestSignature_Ugly(t *core.T) {
	view := axManifestView()
	view.Sign = base64.StdEncoding.EncodeToString([]byte("short"))
	err := resultError(ValidateViewManifestSignature(&view))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not ed25519-sized")
}

func TestManifest_VerifyViewManifestSignature_Good(t *core.T) {
	view, pub := axSignedView(t)
	err := resultError(VerifyViewManifestSignature(&view, pub))
	core.AssertNoError(t, err)
}

func TestManifest_VerifyViewManifestSignature_Bad(t *core.T) {
	view, _ := axSignedView(t)
	wrong, _, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	err = resultError(VerifyViewManifestSignature(&view, wrong))
	core.AssertError(t, err)
}

func TestManifest_VerifyViewManifestSignature_Ugly(t *core.T) {
	view, _ := axSignedView(t)
	err := resultError(VerifyViewManifestSignature(&view, ed25519.PublicKey("short")))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "not an ed25519 public key")
}

func TestManifest_SignViewManifest_Good(t *core.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	view := axManifestView()
	err = resultError(SignViewManifest(&view, priv))
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, view.Sign)
}

func TestManifest_SignViewManifest_Bad(t *core.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	err = resultError(SignViewManifest(nil, priv))
	core.AssertError(t, err)
}

func TestManifest_SignViewManifest_Ugly(t *core.T) {
	view := axManifestView()
	err := resultError(SignViewManifest(&view, ed25519.PrivateKey("short")))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "private key")
}

func TestManifest_CanonicalPackageManifestBytes_Good(t *core.T) {
	pkg := axManifestPackage()
	pkg.Sign = "signature"
	body, err := bytesResult(CanonicalPackageManifestBytes(&pkg))
	core.AssertNoError(t, err)
	core.AssertNotContains(t, string(body), "signature")
}

func TestManifest_CanonicalPackageManifestBytes_Bad(t *core.T) {
	body, err := bytesResult(CanonicalPackageManifestBytes(nil))
	core.AssertNoError(t, err)
	core.AssertContains(t, string(body), "null")
}

func TestManifest_CanonicalPackageManifestBytes_Ugly(t *core.T) {
	pkg := axManifestPackage()
	pkg.Sign = manifestTestKeepMe
	_, err := bytesResult(CanonicalPackageManifestBytes(&pkg))
	core.AssertNoError(t, err)
	core.AssertEqual(t, manifestTestKeepMe, pkg.Sign)
}

func TestManifest_SignPackageManifest_Good(t *core.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	pkg := axManifestPackage()
	err = resultError(SignPackageManifest(&pkg, priv))
	core.AssertNoError(t, err)
	core.AssertNotEmpty(t, pkg.SignKey)
}

func TestManifest_SignPackageManifest_Bad(t *core.T) {
	_, priv, err := ed25519.GenerateKey(nil)
	core.RequireNoError(t, err)
	err = resultError(SignPackageManifest(nil, priv))
	core.AssertError(t, err)
}

func TestManifest_SignPackageManifest_Ugly(t *core.T) {
	pkg := axManifestPackage()
	err := resultError(SignPackageManifest(&pkg, ed25519.PrivateKey("short")))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "private key")
}

func TestManifest_VerifyPackageManifest_Good(t *core.T) {
	pkg, pub := axSignedPackage(t)
	setManifestTrustKeys(t, hex.EncodeToString(pub))
	err := resultError(VerifyPackageManifest(&pkg))
	core.AssertNoError(t, err)
}

func TestManifest_VerifyPackageManifest_Bad(t *core.T) {
	pkg := axManifestPackage()
	err := resultError(VerifyPackageManifest(&pkg))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "unsigned")
}

func TestManifest_VerifyPackageManifest_Ugly(t *core.T) {
	pkg := axManifestPackage()
	pkg.Sign = base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))
	pkg.SignKey = manifestTestNotHex
	err := resultError(VerifyPackageManifest(&pkg))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), manifestTestDecodePackageSignKeyFailed)
}

// TestManifest_buildTargetFromYAML_Good covers the two well-formed shapes:
// the "os/arch" string shorthand and the string-keyed mapping form yaml.v3
// produces for `- os: linux\n  arch: amd64`.
func TestManifest_buildTargetFromYAML_Good(t *core.T) {
	fromString := requireResultValue[BuildTarget](t, buildTargetFromYAML("linux/amd64"))
	core.AssertEqual(t, "linux", fromString.OS)
	core.AssertEqual(t, "amd64", fromString.Arch)

	fromMap := requireResultValue[BuildTarget](t, buildTargetFromYAML(map[string]any{
		manifestTargetOSKey: "darwin",
		"arch":              "arm64",
	}))
	core.AssertEqual(t, "darwin", fromMap.OS)
	core.AssertEqual(t, "arm64", fromMap.Arch)
}

// TestManifest_buildTargetFromYAML_AnyKeyMap covers the defensive map[any]any
// branch (non-string-keyed mappings) routed through stringFromAnyYAMLMap.
func TestManifest_buildTargetFromYAML_AnyKeyMap(t *core.T) {
	target := requireResultValue[BuildTarget](t, buildTargetFromYAML(map[any]any{
		manifestTargetOSKey: "windows",
		"arch":              "amd64",
	}))
	core.AssertEqual(t, "windows", target.OS)
	core.AssertEqual(t, "amd64", target.Arch)

	// Non-string values in the any-keyed map degrade to empty, not a panic.
	partial := requireResultValue[BuildTarget](t, buildTargetFromYAML(map[any]any{
		manifestTargetOSKey: 42,
		"missing":           "ignored",
	}))
	core.AssertEqual(t, "", partial.OS)
	core.AssertEqual(t, "", partial.Arch)
}

// TestManifest_buildTargetFromYAML_Bad covers empty-string, nil and the
// unsupported-type default branch.
func TestManifest_buildTargetFromYAML_Bad(t *core.T) {
	empty := requireResultValue[BuildTarget](t, buildTargetFromYAML(""))
	core.AssertEqual(t, "", empty.OS)

	nilTarget := requireResultValue[BuildTarget](t, buildTargetFromYAML(nil))
	core.AssertEqual(t, "", nilTarget.OS)

	err := resultError(buildTargetFromYAML(42))
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid target entry")
}

// TestManifest_buildTargetFromYAML_Ugly covers malformed string shorthand
// (missing the "/" separator or an empty half).
func TestManifest_buildTargetFromYAML_Ugly(t *core.T) {
	for _, raw := range []string{"linuxonly", "/amd64", "linux/"} {
		err := resultError(buildTargetFromYAML(raw))
		core.AssertError(t, err)
		core.AssertContains(t, err.Error(), "invalid target shorthand")
	}
}

// TestManifest_stringFromAnyYAMLMap_Good asserts a string value at the key is
// returned verbatim.
func TestManifest_stringFromAnyYAMLMap_Good(t *core.T) {
	core.AssertEqual(t, "linux", stringFromAnyYAMLMap(map[any]any{manifestTargetOSKey: "linux"}, manifestTargetOSKey))
}

// TestManifest_stringFromAnyYAMLMap_Bad asserts a missing key and a non-string
// value both yield the empty string.
func TestManifest_stringFromAnyYAMLMap_Bad(t *core.T) {
	core.AssertEqual(t, "", stringFromAnyYAMLMap(map[any]any{}, manifestTargetOSKey))
	core.AssertEqual(t, "", stringFromAnyYAMLMap(map[any]any{manifestTargetOSKey: 7}, manifestTargetOSKey))
}

// TestManifest_stringFromYAMLMap_GoodBad asserts the string-keyed helper returns
// the value when present and a string, and empty otherwise.
func TestManifest_stringFromYAMLMap_GoodBad(t *core.T) {
	core.AssertEqual(t, "amd64", stringFromYAMLMap(map[string]any{"arch": "amd64"}, "arch"))
	core.AssertEqual(t, "", stringFromYAMLMap(map[string]any{"arch": 64}, "arch"))
	core.AssertEqual(t, "", stringFromYAMLMap(map[string]any{}, "arch"))
}

// TestManifest_buildLDFlagsFromYAML_Good covers nil, empty-string, single
// string, []string and []any-of-strings — each a valid ldflags shape.
func TestManifest_buildLDFlagsFromYAML_Good(t *core.T) {
	core.AssertEqual(t, "", requireResultValue[buildmanifestldflags](t, buildLDFlagsFromYAML(nil)).String())
	core.AssertEqual(t, "", requireResultValue[buildmanifestldflags](t, buildLDFlagsFromYAML("")).String())
	core.AssertEqual(t, "-s -w", requireResultValue[buildmanifestldflags](t, buildLDFlagsFromYAML("-s -w")).String())
	core.AssertEqual(t, "-s -w", requireResultValue[buildmanifestldflags](t, buildLDFlagsFromYAML([]string{"-s", "-w"})).String())
	core.AssertEqual(t, "-s -w", requireResultValue[buildmanifestldflags](t, buildLDFlagsFromYAML([]any{"-s", "-w"})).String())
}

// TestManifest_buildLDFlagsFromYAML_Bad covers the []any non-string element, the
// mapping form, and an unsupported scalar type — all rejected.
func TestManifest_buildLDFlagsFromYAML_Bad(t *core.T) {
	core.AssertContains(t, resultError(buildLDFlagsFromYAML([]any{"-s", 7})).Error(), "invalid ldflags sequence")
	core.AssertContains(t, resultError(buildLDFlagsFromYAML(map[string]any{"k": "v"})).Error(), "unsupported ldflags mapping")
	core.AssertContains(t, resultError(buildLDFlagsFromYAML(42)).Error(), "invalid ldflags value")
}

// TestManifest_viewVersionFromYAML_Good covers every accepted scalar form YAML
// can decode a version into: nil, string, int, int64 and float64.
func TestManifest_viewVersionFromYAML_Good(t *core.T) {
	core.AssertEqual(t, ViewVersion(""), requireResultValue[ViewVersion](t, viewVersionFromYAML(nil)))
	core.AssertEqual(t, ViewVersion("1.2.0"), requireResultValue[ViewVersion](t, viewVersionFromYAML("1.2.0")))
	core.AssertEqual(t, ViewVersion("3"), requireResultValue[ViewVersion](t, viewVersionFromYAML(3)))
	core.AssertEqual(t, ViewVersion("9"), requireResultValue[ViewVersion](t, viewVersionFromYAML(int64(9))))
	core.AssertEqual(t, ViewVersion("1.5"), requireResultValue[ViewVersion](t, viewVersionFromYAML(float64(1.5))))
}

// TestManifest_viewVersionFromYAML_Bad asserts an unsupported type is rejected.
func TestManifest_viewVersionFromYAML_Bad(t *core.T) {
	core.AssertContains(t, resultError(viewVersionFromYAML([]string{"nope"})).Error(), "invalid view manifest version")
}
