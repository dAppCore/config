package config

import (
	"embed"

	core "dappco.re/go"
	coreerr "dappco.re/go/log"
	"github.com/xeipuuv/gojsonschema"
)

//go:embed schema/*.schema.json
var schemaFS embed.FS

var manifestSchemas = map[string]string{
	FileConfig:    "schema/config.schema.json",
	FileBuild:     "schema/build.schema.json",
	FileRelease:   "schema/release.schema.json",
	FileTest:      "schema/test.schema.json",
	FileRun:       "schema/run.schema.json",
	FileView:      "schema/view.schema.json",
	FileManifest:  "schema/manifest.schema.json",
	FileWorkspace: "schema/workspace.schema.json",
	FileRepos:     "schema/repos.schema.json",
	FileIDE:       "schema/ide.schema.json",
	FilePHP:       "schema/php.schema.json",
	FileAgent:     "schema/agent.schema.json",
	FileZone:      "schema/zone.schema.json",
}

const callerValidateSchema = "config.validateSchema"

// validateSchema applies an embedded JSON schema when the current filename has
// one. Empty documents are treated as absent config and skip validation so
// blank files remain a non-error baseline.
func validateSchema(path string, raw map[string]any) core.Result {
	if len(raw) == 0 {
		return core.Ok(nil)
	}

	schemaPath, ok := manifestSchemas[core.PathBase(path)]
	if !ok {
		return core.Ok(nil)
	}

	schemaBody, err := schemaFS.ReadFile(schemaPath)
	if err != nil {
		return core.Fail(coreerr.E(callerValidateSchema, "failed to read embedded schema: "+schemaPath, err))
	}

	documentResult := core.JSONMarshal(raw)
	if !documentResult.OK {
		return core.Fail(coreerr.E(callerValidateSchema, "failed to encode config for schema validation: "+path, resultCause(documentResult).(error)))
	}
	documentBody := documentResult.Value.([]byte)

	result, err := gojsonschema.Validate(
		gojsonschema.NewBytesLoader(schemaBody),
		gojsonschema.NewBytesLoader(documentBody),
	)
	if err != nil {
		return core.Fail(coreerr.E(callerValidateSchema, "schema validation failed: "+path, err))
	}
	if result.Valid() {
		return core.Ok(nil)
	}

	var problems []string
	for _, issue := range result.Errors() {
		problems = append(problems, issue.String())
	}
	return core.Fail(coreerr.E(
		callerValidateSchema,
		"schema validation failed: "+path+": "+core.Join("; ", problems...),
		nil,
	))
}
