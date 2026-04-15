package config

import (
	"encoding/json"

	core "dappco.re/go/core"
	coreio "dappco.re/go/core/io"
	coreerr "dappco.re/go/core/log"
)

func detectTestCommand(medium coreio.Medium, start string) (string, bool, error) {
	start = normalizeUpwardStart(medium, start)
	for dir := start; ; dir = core.PathDir(dir) {
		if command, ok, err := detectTestCommandAtDir(medium, dir); err != nil {
			return "", false, err
		} else if ok {
			return command, true, nil
		}

		if medium.Exists(core.Path(dir, ".git")) {
			break
		}
		parent := core.PathDir(dir)
		if parent == dir {
			break
		}
	}

	return "", false, nil
}

func detectTestCommandAtDir(medium coreio.Medium, dir string) (string, bool, error) {
	switch {
	case medium.Exists(core.Path(dir, "composer.json")):
		return detectJSONTestCommand(medium, core.Path(dir, "composer.json"), "composer", "composer test")
	case medium.Exists(core.Path(dir, "package.json")):
		return detectJSONTestCommand(medium, core.Path(dir, "package.json"), "npm", "npm test")
	case medium.Exists(core.Path(dir, "go.mod")):
		return "go test ./...", true, nil
	case medium.Exists(core.Path(dir, "pytest.ini")):
		return "pytest", true, nil
	case medium.Exists(core.Path(dir, "pyproject.toml")):
		return "pytest", true, nil
	case medium.Exists(core.Path(dir, "Taskfile.yaml")) || medium.Exists(core.Path(dir, "Taskfile.yml")):
		return "task test", true, nil
	default:
		return "", false, nil
	}
}

func detectJSONTestCommand(medium coreio.Medium, path string, label string, fallback string) (string, bool, error) {
	content, err := medium.Read(path)
	if err != nil {
		return "", false, coreerr.E("config.ResolveTestManifest", "failed to read "+label+" manifest: "+path, err)
	}

	var data struct {
		Scripts map[string]json.RawMessage `json:"scripts"`
	}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return "", false, coreerr.E("config.ResolveTestManifest", "failed to parse "+label+" manifest: "+path, err)
	}

	raw, ok := data.Scripts["test"]
	if !ok {
		return fallback, true, nil
	}

	var script string
	if err := json.Unmarshal(raw, &script); err != nil {
		return "", false, coreerr.E("config.ResolveTestManifest", "invalid "+label+" test script: "+path, err)
	}
	if script == "" {
		return fallback, true, nil
	}
	return script, true, nil
}
