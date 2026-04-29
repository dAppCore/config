package config

import (
	"crypto/ed25519"
	"encoding/hex"
	"syscall"

	core "dappco.re/go"
	coreio "dappco.re/go/io"
	"gopkg.in/yaml.v3"
)

func exampleSetManifestTrustKey(key string) func() {
	previous, hadPrevious := syscall.Getenv("CORE_MANIFEST_TRUST_KEYS")
	_ = syscall.Setenv("CORE_MANIFEST_TRUST_KEYS", key)
	return func() {
		if hadPrevious {
			_ = syscall.Setenv("CORE_MANIFEST_TRUST_KEYS", previous)
			return
		}
		_ = syscall.Unsetenv("CORE_MANIFEST_TRUST_KEYS")
	}
}

func exampleViewManifest() ViewManifest {
	return ViewManifest{
		Version: ViewVersion("1"),
		Code:    "app",
		Name:    "Example App",
		Layout:  "HLCRF",
		Slots:   map[string]any{"C": "main"},
	}
}

func exampleSignedViewManifest() (ViewManifest, ed25519.PublicKey) {
	pub, priv, _ := ed25519.GenerateKey(nil)
	view := exampleViewManifest()
	_ = SignViewManifest(&view, priv)
	return view, pub
}

func examplePackageManifest() PackageManifest {
	return PackageManifest{
		Code:        "go-config",
		Name:        "Config",
		Module:      "dappco.re/go/config",
		Version:     "1.0.0",
		Description: "Layered configuration",
		Licence:     "EUPL-1.2",
	}
}

func exampleSignedPackageManifest() (PackageManifest, func()) {
	_, priv, _ := ed25519.GenerateKey(nil)
	pkg := examplePackageManifest()
	_ = SignPackageManifest(&pkg, priv)
	return pkg, exampleSetManifestTrustKey(pkg.SignKey)
}

func ExampleViewVersion_UnmarshalYAML() {
	var version ViewVersion
	err := version.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "2"})
	core.Println(err == nil, version)
	// Output: true 2
}

func ExampleBuildTarget_UnmarshalYAML() {
	var target BuildTarget
	err := target.UnmarshalYAML(&yaml.Node{Kind: yaml.ScalarNode, Value: "linux/amd64"})
	core.Println(err == nil, target.OS, target.Arch)
	// Output: true linux amd64
}

func ExampleBuildManifest_UnmarshalYAML() {
	var build BuildManifest
	body := "version: 1\nproject:\n  name: app\n  main: ./cmd/app\nbuild:\n  flags: [-trimpath]\ntargets:\n  - linux/amd64\n"
	err := yaml.Unmarshal([]byte(body), &build)
	core.Println(err == nil, build.Project.Name, build.Targets[0].Arch)
	// Output: true app amd64
}

func ExampleLoadManifest() {
	m := coreio.NewMockMedium()
	path := "/example/.core/build.yaml"
	_ = m.Write(path, "version: 1\nproject:\n  name: app\n")
	var build BuildManifest
	err := LoadManifest(m, path, &build)
	core.Println(err == nil, build.Project.Name)
	// Output: true app
}

func ExampleCanonicalViewManifestBytes() {
	view := exampleViewManifest()
	body, err := CanonicalViewManifestBytes(&view)
	core.Println(err == nil, core.Contains(string(body), "code: app"))
	// Output: true true
}

func ExampleValidateViewManifestSignature() {
	view, _ := exampleSignedViewManifest()
	err := ValidateViewManifestSignature(&view)
	core.Println(err == nil)
	// Output: true
}

func ExampleVerifyViewManifestSignature() {
	view, pub := exampleSignedViewManifest()
	err := VerifyViewManifestSignature(&view, pub)
	core.Println(err == nil)
	// Output: true
}

func ExampleSignViewManifest() {
	_, priv, _ := ed25519.GenerateKey(nil)
	view := exampleViewManifest()
	err := SignViewManifest(&view, priv)
	core.Println(err == nil, view.Sign != "")
	// Output: true true
}

func ExampleCanonicalPackageManifestBytes() {
	pkg := examplePackageManifest()
	body, err := CanonicalPackageManifestBytes(&pkg)
	core.Println(err == nil, core.Contains(string(body), "code: go-config"))
	// Output: true true
}

func ExampleSignPackageManifest() {
	_, priv, _ := ed25519.GenerateKey(nil)
	pkg := examplePackageManifest()
	err := SignPackageManifest(&pkg, priv)
	core.Println(err == nil, pkg.Sign != "", pkg.SignKey != "")
	// Output: true true true
}

func ExampleVerifyPackageManifest() {
	pkg, cleanup := exampleSignedPackageManifest()
	defer cleanup()
	err := VerifyPackageManifest(&pkg)
	core.Println(err == nil)
	// Output: true
}

func ExampleTrustedManifestPublicKeys() {
	pub, _, _ := ed25519.GenerateKey(nil)
	cleanup := exampleSetManifestTrustKey(hex.EncodeToString(pub))
	defer cleanup()
	keys, err := TrustedManifestPublicKeys()
	core.Println(err == nil, len(keys))
	// Output: true 1
}
