# CLAUDE.md

This file gives Claude Code agents repository-specific context for
`dappco.re/go/config`.

## Commands

Use the explicit verification commands from the compliance brief:

```bash
GOWORK=off go mod tidy
GOWORK=off go vet ./...
GOWORK=off go test -count=1 ./...
gofmt -l .
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```

The audit script is the completion contract. The repository is not complete
until every audit counter is `0` and the verdict is `COMPLIANT`.

## Architecture

`Config` uses two Viper instances:

- `file` stores values loaded from config files and values explicitly set at
  runtime. This is the only source written by `Commit`.
- `full` is the read view. It is rebuilt from file settings, environment
  variables, and explicit overrides.

This shape keeps environment-derived secrets out of persisted YAML while still
letting `Get` and `All` see the effective runtime configuration.

## API Conventions

This consumer repo follows `dappco.re/go` v0.9.0 conventions:

- Public operations return `core.Result`.
- Callers branch on `r.OK` and read `r.Value` or `r.Error()`.
- Use `core.Ok`, `core.Fail`, and `core.ResultOf` rather than struct literals.
- Do not import banned stdlib packages directly. Reach through Core wrappers
  such as `core.PathExt`, `core.PathDir`, `core.Environ`, `core.SplitN`,
  `core.Replace`, `core.NewReader`, and `core.Sprintf`.

## Tests And Examples

Each production file with public symbols has a sibling test file and a sibling
example file. Keep coverage file-aware:

- `config.go` -> `config_test.go` and `config_example_test.go`
- `env.go` -> `env_test.go` and `env_example_test.go`
- `service.go` -> `service_test.go` and `service_example_test.go`

Tests use `*core.T`, dot-import `dappco.re/go`, and the Core assertion helpers.
Triplets are named `Test<File>_<Symbol>_{Good,Bad,Ugly}`. Examples use
`core.Println`, not `fmt.Println`.

## Do Not Touch

Do not edit `BRIEF.md`, `.git`, `.codex`, or any `third_party` directory.
