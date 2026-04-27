package config

import (
	"path/filepath"
	"testing"

	coreio "dappco.re/go/io"
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
			expected: "core go qa",
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
		{
			name:     "taskfile-yml",
			filename: "Taskfile.yml",
			content:  "version: '3'\n",
			expected: "task test",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m := coreio.NewMockMedium()
			root := filepath.Join("/", "repo", tc.name)
			child := filepath.Join(root, "service")

			assert.NoError(t, m.EnsureDir(filepath.Join(root, ".core")))
			assert.NoError(t, m.EnsureDir(child))
			assert.NoError(t, m.EnsureDir(filepath.Join(root, ".git")))

			path := filepath.Join(root, tc.filename)
			if tc.filename == FileTest {
				path = filepath.Join(root, ".core", FileTest)
			}
			assert.NoError(t, m.Write(path, tc.content))

			manifest, err := ResolveTestManifest(m, child)
			assert.NoError(t, err)
			assert.NotNil(t, manifest)
			assert.NotEmpty(t, manifest.Commands)
			assert.Equal(t, tc.expected, manifest.Commands[0].Run)
		})
	}
}

func TestTestDetect_ResolveTestManifest_Bad(t *testing.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "repo", "bad")
	child := filepath.Join(root, "service")

	assert.NoError(t, m.EnsureDir(filepath.Join(root, ".git")))
	assert.NoError(t, m.EnsureDir(child))
	assert.NoError(t, m.Write(filepath.Join(root, "package.json"), `{"scripts":{"test":123}}`))

	manifest, err := ResolveTestManifest(m, child)
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid npm test script")
}

func TestTestDetect_ResolveTestManifest_Ugly(t *testing.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "repo", "ugly")

	assert.NoError(t, m.EnsureDir(filepath.Join(root, ".git")))

	manifest, err := ResolveTestManifest(m, root)
	assert.Nil(t, manifest)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no test command could be detected")
}
