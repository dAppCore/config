package config

import (
	core "dappco.re/go"
	"path/filepath"

	coreio "dappco.re/go/io"
)

func TestTestDetect_ResolveTestManifest_Good(t *core.T) {
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
		t.Run(tc.name, func(t *core.T) {
			m := coreio.NewMockMedium()
			root := filepath.Join("/", "repo", tc.name)
			child := filepath.Join(root, "service")

			core.AssertNoError(t, m.EnsureDir(filepath.Join(root, ".core")))
			core.AssertNoError(t, m.EnsureDir(child))
			core.AssertNoError(t, m.EnsureDir(filepath.Join(root, ".git")))

			path := filepath.Join(root, tc.filename)
			if tc.filename == FileTest {
				path = filepath.Join(root, ".core", FileTest)
			}
			core.AssertNoError(t, m.Write(path, tc.content))

			manifest, err := ResolveTestManifest(m, child)
			core.AssertNoError(t, err)
			core.AssertNotNil(t, manifest)
			core.AssertNotEmpty(t, manifest.Commands)
			core.AssertEqual(t, tc.expected, manifest.Commands[0].Run)
		})
	}
}

func TestTestDetect_ResolveTestManifest_Bad(t *core.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "repo", "bad")
	child := filepath.Join(root, "service")

	core.AssertNoError(t, m.EnsureDir(filepath.Join(root, ".git")))
	core.AssertNoError(t, m.EnsureDir(child))
	core.AssertNoError(t, m.Write(filepath.Join(root, "package.json"), `{"scripts":{"test":123}}`))

	manifest, err := ResolveTestManifest(m, child)
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "invalid npm test script")
}

func TestTestDetect_ResolveTestManifest_Ugly(t *core.T) {
	m := coreio.NewMockMedium()
	root := filepath.Join("/", "repo", "ugly")

	core.AssertNoError(t, m.EnsureDir(filepath.Join(root, ".git")))

	manifest, err := ResolveTestManifest(m, root)
	core.AssertNil(t, manifest)
	core.AssertError(t, err)
	core.AssertContains(t, err.Error(), "no test command could be detected")
}
