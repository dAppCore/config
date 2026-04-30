# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repo Layout

```text
core/config/
├── go/
│   ├── *.go                      ← Go source files moved from repository root
│   ├── tests/                    ← Go CLI tests moved from repository root
│   ├── go.mod
│   ├── go.sum
│   ├── README.md -> ../README.md  ← symlink
│   ├── CLAUDE.md -> ../CLAUDE.md  ← symlink
│   ├── AGENTS.md -> ../AGENTS.md  ← symlink
│   └── docs -> ../docs            ← symlink
├── docs/
├── schema/
├── README.md
├── CLAUDE.md
├── AGENTS.md
├── LICENSE/
├── SONAR-project files
└── other non-Go/cross-language assets
```

The Go module path remains `dappco.re/go/config`; only repository layout changed.

## Go Resolution Modes

This repository is module-local under `go/` and does not use a local workspace file in this module root. Consumer commands should either:

1. Run from `go/` directly.
2. Prefix Go commands with `cd go &&` when executed from repository root.

Examples:

```bash
cd go && go test ./...
cd go && golangci-lint run ./...
cd go && core go qa
```

For CI-style reproducible builds, force module-only behavior:

```bash
GOWORK=off go test ./...
GOFLAGS=-mod=mod go vet ./...
```

## Build & Test Commands

This project uses the Core CLI (`core` binary), not `go` directly.

```bash
cd go && go test ./...                                    # run all tests
cd go && GOWORK=off go test -run TestConfig_Get_Good ./...  # run a single test
cd go && GOWORK=off go test -cover ./...                   # test with coverage

cd go && core go qa                            # format, vet, lint, test
cd go && core go qa full                       # adds race detector, vuln scan, security audit

cd go && core go fmt                           # format
cd go && core go vet                           # vet
cd go && core go lint                          # lint
```

This is a library package — there is no binary to build or run.

## Architecture

**Dual-Viper pattern**: `Config` holds two `*viper.Viper` instances:
- `v` (full) — file + env + defaults; used for all reads (`Get`, `All`)
- `f` (file-only) — file + explicit `Set()` calls; used for persistence (`Commit`)

This prevents environment variables from leaking into saved config files. When implementing new features, maintain this invariant: writes go to both `v` and `f`; reads come from `v`; persistence comes from `f`.

**Resolution priority** (ascending): defaults → file → env vars (`CORE_CONFIG_*`) → `Set()`

**Service wrapper**: `Service` in `service.go` wraps `Config` with framework lifecycle (`core.Startable`). Both `Config` and `Service` satisfy `core.Config`, enforced by compile-time assertions.

**Storage abstraction**: All file I/O goes through `coreio.Medium` (from `go-io`). Tests use `coreio.NewMockMedium()` with an in-memory `Files` map — never touch the real filesystem.

## Conventions

- **UK English** in comments and documentation (colour, organisation, centre)
- **Error wrapping**: `coreerr.E(caller, message, underlying)` from `go-log`
- **Test naming**: `_Good` (happy path), `_Bad` (expected errors), `_Ugly` (panics/edge cases)
- **Functional options**: `New()` takes `...Option` (e.g. `WithMedium`, `WithPath`, `WithEnvPrefix`)
- **Conventional commits**: `type(scope): description`
- **Go workspace**: no per-repo `go.work`; this module resolves with `GOWORK=off`

## Dependencies

- `dappco.re/go/core/io` — `Medium` interface for storage
- `dappco.re/go/core/log` — `coreerr.E()` error helper
- `dappco.re/go/core` — `core.Core`, `core.Startable`, `core.ServiceRuntime`, primitives
- `github.com/spf13/viper` — configuration engine
- `github.com/stretchr/testify` — test assertions
