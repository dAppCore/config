package config

import (
	"path/filepath"
	"testing"

	coreio "dappco.re/go/core/io"
	"github.com/stretchr/testify/assert"
)

func TestTestDetect_ResolveTestManifest_Good(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		content  string
		expected string
	}{
		{
			name:     "core manifest",
			filename: FileTest,
			content:  "version: 1\ncommands:\n  - name: unit\n    run: vendor/bin/pest --parallel\n",
			expected: "vendor/bin/pest --parallel",
		},
		{
			name:     "composer fallback",
			filename: "composer.json",
			content:  `{"scripts":{}}`,
			expected: "composer test",
		},
		{
			name:     "package script",
			filename: "package.json",
			content:  `{"scripts":{"test":"npm run test:unit"}}`,
			expected: "npm run test:unit",
		},
		{
			name:     "go module",
			filename: "go.mod",
			content:  "module example.com/repo\n",
			expected: "go test ./...",
		},
		{
			name:     "pytest ini",
			filename: "pytest.ini",
			content:  "[pytest]\n",
			expected: "pytest",
		},
		{
			name:     "pyproject",
			filename: "pyproject.toml",
			content:  "[tool.pytest.ini_options]\n",
			expected: "pytest",
		},
		{
			name:     "taskfile",
			filename: "Taskfile.yaml",
			content:  "version: '3'\n",
			expected: "task test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmp := t.TempDir()
			root := filepath.Join(tmp, "repo")
			child := filepath.Join(root, "service")

			assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(root, ".core")))
			assert.NoError(t, coreio.Local.EnsureDir(child))
			assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(root, ".git")))

			path := filepath.Join(root, tc.filename)
			if tc.filename == FileTest {
				path = filepath.Join(root, ".core", FileTest)
			}
			assert.NoError(t, coreio.Local.Write(path, tc.content))

			manifest, err := ResolveTestManifest(coreio.Local, child)
			assert.NoError(t, err)
			assert.NotNil(t, manifest)
			assert.NotEmpty(t, manifest.Commands)
			assert.Equal(t, tc.expected, manifest.Commands[0].Run)
		})
	}
}

func TestTestDetect_ResolveTestManifest_Bad(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")
	child := filepath.Join(root, "service")

	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(root, ".git")))
	assert.NoError(t, coreio.Local.EnsureDir(child))
	assert.NoError(t, coreio.Local.Write(filepath.Join(root, "package.json"), `{"scripts":{"test":123}}`))

	manifest, err := ResolveTestManifest(coreio.Local, child)
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid npm test script")
}

func TestTestDetect_ResolveTestManifest_Ugly(t *testing.T) {
	tmp := t.TempDir()
	root := filepath.Join(tmp, "repo")

	assert.NoError(t, coreio.Local.EnsureDir(filepath.Join(root, ".git")))

	manifest, err := ResolveTestManifest(coreio.Local, root)
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no test command could be detected")
}
