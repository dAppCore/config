package config

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"path/filepath"
	"testing"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

type falseExistsMedium struct {
	*coreio.MockMedium
}

func (m falseExistsMedium) Exists(string) bool {
	return false
}

func TestResolve_FindConfigManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	base := filepath.Join(tmp, "workspace")
	repo := filepath.Join(base, "repo")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(base, ".core"),
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".git"),
		child,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	globalConfig := filepath.Join(core.Env("DIR_HOME"), ".core", FileConfig)
	projectConfig := filepath.Join(repo, ".core", FileConfig)
	workspaceRepos := filepath.Join(base, ".core", FileRepos)
	assert.NoError(t, m.Write(globalConfig, "app:\n  name: global\n"))
	assert.NoError(t, m.Write(projectConfig, "app:\n  name: project\n"))
	assert.NoError(t, m.Write(workspaceRepos, "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	assert.Equal(t, projectConfig, FindConfigManifest(m, child))
}

func TestResolve_FindConfigManifest_Bad(t *testing.T) {
	m := falseExistsMedium{coreio.NewMockMedium()}

	assert.Empty(t, FindConfigManifest(m, t.TempDir()))
}

func TestResolve_FindConfigManifest_Ugly(t *testing.T) {
	m := falseExistsMedium{coreio.NewMockMedium()}
	start := filepath.Join(t.TempDir(), "missing", "service")

	assert.Empty(t, FindConfigManifest(m, start))
}

func TestResolve_FindProjectManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	base := filepath.Join(tmp, "workspace")
	repo := filepath.Join(base, "repo")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(base, ".core"),
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".git"),
		child,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}
	assert.NoError(t, m.EnsureDir(filepath.Join(base, ".core")))

	files := map[string]string{
		FileBuild:     "name: core\noutput: dist\ntargets:\n  - linux/amd64\n",
		FileRelease:   "version: 1\narchive:\n  format: tar.gz\n  include:\n    - README.md\nchecksums: true\ngithub:\n  draft: false\n  prerelease: false\nchangelog:\n  include:\n    - feat\n",
		FileRun:       "version: 1\nservices:\n  - name: db\n    image: postgres:16\n    port: 5432\ndev:\n  command: php artisan serve\n  port: 8000\n  watch:\n    - app/\n",
		FileView:      "code: photo-browser\nname: Photo Browser\nsign: " + base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize)) + "\npermissions:\n  clipboard: true\n",
		FileManifest:  packageManifestFixture(t),
		FileWorkspace: "version: 1\ndependencies:\n  - core-php\nactive: core-php\npackages_dir: ./packages\n",
	}

	for name, content := range files {
		assert.NoError(t, m.Write(filepath.Join(repo, ".core", name), content))
	}
	assert.NoError(t, m.Write(filepath.Join(base, ".core", FileBuild), "name: external\noutput: ext\n"))
	assert.NoError(t, m.Write(filepath.Join(base, ".core", FileRepos), "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	cases := []struct {
		name string
		path string
		got  string
	}{
		{name: "build", path: filepath.Join(repo, ".core", FileBuild), got: FindBuildManifest(m, child)},
		{name: "release", path: filepath.Join(repo, ".core", FileRelease), got: FindReleaseManifest(m, child)},
		{name: "run", path: filepath.Join(repo, ".core", FileRun), got: FindRunManifest(m, child)},
		{name: "view", path: filepath.Join(repo, ".core", FileView), got: FindViewManifest(m, child)},
		{name: "manifest", path: filepath.Join(repo, ".core", FileManifest), got: FindPackageManifest(m, child)},
		{name: "workspace", path: filepath.Join(repo, ".core", FileWorkspace), got: FindWorkspaceManifest(m, child)},
		{name: "repos", path: filepath.Join(base, ".core", FileRepos), got: FindReposManifest(m, child)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.path, tc.got)
		})
	}
}

func TestResolve_FindProjectManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	base := filepath.Join(tmp, "workspace")
	repo := filepath.Join(base, "repo")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(base, ".core"),
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".git"),
		child,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	assert.Empty(t, FindBuildManifest(m, child))
	assert.Empty(t, FindReleaseManifest(m, child))
	assert.Empty(t, FindRunManifest(m, child))
	assert.Empty(t, FindViewManifest(m, child))
	assert.Empty(t, FindPackageManifest(m, child))
	assert.Empty(t, FindReposManifest(m, child))
	assert.Empty(t, FindWorkspaceManifest(m, child))
}

func TestResolve_FindProjectManifest_Ugly(t *testing.T) {
	// Nil medium falls back to the local filesystem; with no .core tree the
	// project-local wrappers should still return an empty path instead of panicking.
	// Repos are resolved from the shared workspace root and may legitimately
	// exist on the host machine, so this test does not assert on repos.yaml.
	start := filepath.Join(t.TempDir(), "missing", "service")
	assert.Empty(t, FindBuildManifest(nil, start))
	assert.Empty(t, FindReleaseManifest(nil, start))
	assert.Empty(t, FindRunManifest(nil, start))
	assert.Empty(t, FindViewManifest(nil, start))
	assert.Empty(t, FindPackageManifest(nil, start))
}

func TestResolve_ResolveConfigManifest_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := filepath.Join(tmp, "workspace", "repo", "service")

	for _, dir := range []string{
		filepath.Join(home, ".core"),
		start,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	configPath := filepath.Join(home, ".core", FileConfig)
	assert.NoError(t, m.Write(configPath, "app:\n  name: global\n"))

	cfg, err := ResolveConfigManifest(m, start)
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, configPath, cfg.Path())

	var name string
	assert.NoError(t, cfg.Get("app.name", &name))
	assert.Equal(t, "global", name)
}

func TestResolve_ResolveConfigManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	start := filepath.Join(t.TempDir(), "workspace", "repo", "service")

	_, err := ResolveConfigManifest(m, start)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no config manifest could be detected")
}

func TestResolve_ResolveConfigManifest_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	home := core.Env("DIR_HOME")
	start := filepath.Join(tmp, "workspace", "repo", "service")

	for _, dir := range []string{
		filepath.Join(home, ".core"),
		start,
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	configPath := filepath.Join(home, ".core", FileConfig)
	assert.NoError(t, m.Write(configPath, "app:\n  name: [broken yaml"))

	_, err := ResolveConfigManifest(m, start)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load config file")
}

func TestResolve_ResolveProjectManifests_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "workspace", "repo")
	workspace := filepath.Join(tmp, "workspace")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".git"),
		child,
		filepath.Join(workspace, ".core"),
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	buildPath := filepath.Join(repo, ".core", FileBuild)
	releasePath := filepath.Join(repo, ".core", FileRelease)
	runPath := filepath.Join(repo, ".core", FileRun)
	viewPath := filepath.Join(repo, ".core", FileView)
	packagePath := filepath.Join(repo, ".core", FileManifest)

	assert.NoError(t, m.Write(buildPath, "name: core\noutput: dist\ntargets:\n  - linux/amd64\n"))
	assert.NoError(t, m.Write(releasePath, "version: 1\narchive:\n  format: tar.gz\n  include:\n    - README.md\nchecksums: true\ngithub:\n  draft: false\n  prerelease: false\nchangelog:\n  include:\n    - feat\n"))
	assert.NoError(t, m.Write(runPath, "version: 1\nservices:\n  - name: db\n    image: postgres:16\n    port: 5432\ndev:\n  command: php artisan serve\n  port: 8000\n  watch:\n    - app/\n"))
	assert.NoError(t, m.Write(viewPath, "code: photo-browser\nname: Photo Browser\nsign: "+base64.StdEncoding.EncodeToString(make([]byte, ed25519.SignatureSize))+"\npermissions:\n  clipboard: true\n"))
	assert.NoError(t, m.Write(packagePath, packageManifestFixture(t)))
	assert.NoError(t, m.Write(filepath.Join(workspace, ".core", FileRepos), "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	build, err := ResolveBuildManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, "core", build.Name)
	assert.Equal(t, "dist", build.Output)

	release, err := ResolveReleaseManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, true, release.Checksums)
	assert.Equal(t, "tar.gz", release.Archive.Format)

	run, err := ResolveRunManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, "php artisan serve", run.Dev.Command)
	assert.Len(t, run.Services, 1)

	view, err := ResolveViewManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, "photo-browser", view.Code)
	assert.True(t, view.Permissions.Clipboard)

	pkg, err := ResolvePackageManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, "go-io", pkg.Code)
	assert.Equal(t, "Core I/O", pkg.Name)

	repos, err := ResolveReposManifest(m, child)
	assert.NoError(t, err)
	assert.Equal(t, "host-uk", repos.Org)
	assert.Len(t, repos.Repos, 1)
}

func TestResolve_ResolveProjectManifests_Bad(t *testing.T) {
	t.Setenv("DIR_HOME", t.TempDir())
	m := coreio.NewMockMedium()
	start := filepath.Join(t.TempDir(), "workspace", "repo", "service")

	_, err := ResolveBuildManifest(m, start)
	assert.Error(t, err)
	_, err = ResolveReleaseManifest(m, start)
	assert.Error(t, err)
	_, err = ResolveRunManifest(m, start)
	assert.Error(t, err)
	_, err = ResolveViewManifest(m, start)
	assert.Error(t, err)
	_, err = ResolvePackageManifest(m, start)
	assert.Error(t, err)
	_, err = ResolveReposManifest(m, start)
	assert.Error(t, err)
}

func TestResolve_FindReposManifest_FallsBackToWorkspaceRoot_Good(t *testing.T) {
	m := coreio.NewMockMedium()
	start := filepath.Join(t.TempDir(), "workspace", "repo", "service")
	reposPath := filepath.Join(core.Env("DIR_HOME"), "Code", ".core", FileRepos)

	assert.NoError(t, m.EnsureDir(filepath.Dir(reposPath)))
	assert.NoError(t, m.Write(reposPath, "version: 1\norg: host-uk\nrepos:\n  - path: core/go\n    remote: ssh://forge.example/core/go.git\n"))

	assert.Equal(t, reposPath, FindReposManifest(m, start))
}

func TestResolve_ResolveProjectManifests_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	tmp := t.TempDir()
	repo := filepath.Join(tmp, "workspace", "repo")
	workspace := filepath.Join(tmp, "workspace")
	child := filepath.Join(repo, "service")

	for _, dir := range []string{
		filepath.Join(repo, ".core"),
		filepath.Join(repo, ".git"),
		child,
		filepath.Join(workspace, ".core"),
	} {
		assert.NoError(t, m.EnsureDir(dir))
	}

	assert.NoError(t, m.Write(filepath.Join(repo, ".core", FileBuild), "name: core\noutput: dist\ntargets:\n  - [broken yaml"))
	assert.NoError(t, m.Write(filepath.Join(repo, ".core", FileRelease), "version: 1\narchive:\n  format: [broken yaml"))
	assert.NoError(t, m.Write(filepath.Join(repo, ".core", FileRun), "version: 1\nservices: [broken yaml"))
	assert.NoError(t, m.Write(filepath.Join(repo, ".core", FileView), "code: photo-browser\nname: Photo Browser\npermissions:\n  clipboard: true\n"))
	assert.NoError(t, m.Write(filepath.Join(repo, ".core", FileManifest), "code: go-io\nname: Core I/O\nsign: not-base64\nsign_key: \"\"\n"))
	assert.NoError(t, m.Write(filepath.Join(workspace, ".core", FileRepos), "version: 1\norg: host-uk\nrepos: [broken yaml"))

	_, err := ResolveBuildManifest(m, child)
	assert.Error(t, err)
	_, err = ResolveReleaseManifest(m, child)
	assert.Error(t, err)
	_, err = ResolveRunManifest(m, child)
	assert.Error(t, err)

	_, err = ResolveViewManifest(m, child)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsigned view manifest rejected")

	_, err = ResolvePackageManifest(m, child)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "missing package sign_key")

	_, err = ResolveReposManifest(m, child)
	assert.Error(t, err)
}

func packageManifestFixture(t *testing.T) string {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(nil)
	assert.NoError(t, err)

	pkg := &PackageManifest{
		Code:    "go-io",
		Name:    "Core I/O",
		Version: "0.3.0",
		Licence: "EUPL-1.2",
		SignKey: hex.EncodeToString(pub),
	}
	msg, err := packageManifestBytes(pkg)
	assert.NoError(t, err)
	pkg.Sign = base64.StdEncoding.EncodeToString(ed25519.Sign(priv, msg))

	out, err := yaml.Marshal(pkg)
	assert.NoError(t, err)
	return string(out)
}
