package config

import (
	core "dappco.re/go"
	coreio "dappco.re/go/io"
	coreerr "dappco.re/go/log"
)

const callerResolveTestManifest = "config.ResolveTestManifest"

type detectedTestCommand struct {
	Command string
	Found   bool
}

func detectTestCommand(medium coreio.Medium, start string) core.Result {
	start = normalizeUpwardStart(medium, start)
	for dir := start; ; dir = core.PathDir(dir) {
		detectedResult := detectTestCommandAtDir(medium, dir)
		if !detectedResult.OK {
			return detectedResult
		}
		detected := detectedResult.Value.(detectedTestCommand)
		if detected.Found {
			return detectedResult
		}

		if medium.Exists(core.Path(dir, ".git")) {
			break
		}
		parent := core.PathDir(dir)
		if parent == dir {
			break
		}
	}

	return core.Ok(detectedTestCommand{})
}

func detectTestCommandAtDir(medium coreio.Medium, dir string) core.Result {
	switch {
	case medium.Exists(core.Path(dir, "composer.json")):
		return detectJSONTestCommand(medium, core.Path(dir, "composer.json"), "composer", "composer test")
	case medium.Exists(core.Path(dir, "package.json")):
		return detectJSONTestCommand(medium, core.Path(dir, "package.json"), "npm", "npm test")
	case medium.Exists(core.Path(dir, "go.mod")):
		return core.Ok(detectedTestCommand{Command: "core go qa", Found: true})
	case medium.Exists(core.Path(dir, "pytest.ini")):
		return core.Ok(detectedTestCommand{Command: "pytest", Found: true})
	case medium.Exists(core.Path(dir, "pyproject.toml")):
		return core.Ok(detectedTestCommand{Command: "pytest", Found: true})
	case medium.Exists(core.Path(dir, "Taskfile.yaml")) || medium.Exists(core.Path(dir, "Taskfile.yml")):
		return core.Ok(detectedTestCommand{Command: "task test", Found: true})
	default:
		return core.Ok(detectedTestCommand{})
	}
}

func detectJSONTestCommand(medium coreio.Medium, path, label, fallback string) core.Result {
	content, err := medium.Read(path)
	if err != nil {
		return core.Fail(coreerr.E(callerResolveTestManifest, "failed to read "+label+" manifest: "+path, err))
	}

	var data struct {
		Scripts map[string]any `json:"scripts"`
	}
	if r := core.JSONUnmarshalString(content, &data); !r.OK {
		return core.Fail(coreerr.E(callerResolveTestManifest, "failed to parse "+label+" manifest: "+path, resultCause(r).(error)))
	}

	raw, ok := data.Scripts["test"]
	if !ok {
		return core.Ok(detectedTestCommand{Command: fallback, Found: true})
	}

	script, ok := raw.(string)
	if !ok {
		return core.Fail(coreerr.E(callerResolveTestManifest, "invalid "+label+" test script: "+path, nil))
	}
	if script == "" {
		return core.Ok(detectedTestCommand{Command: fallback, Found: true})
	}
	return core.Ok(detectedTestCommand{Command: script, Found: true})
}
