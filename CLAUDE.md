# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

This project uses the Core CLI (`core` binary), not `go` directly.

```bash
go test ./...                                    # run all tests
go test -run TestConfig_Get_Good ./...           # run a single test
go test -cover ./...                             # test with coverage

core go qa                            # format, vet, lint, test
core go qa full                       # adds race detector, vuln scan, security audit

core go fmt                           # format
core go vet                           # vet
core go lint                          # lint
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
- **Go workspace**: module is part of `~/Code/go.work`

## Dependencies

- `forge.lthn.ai/core/go-io` — `Medium` interface for storage
- `forge.lthn.ai/core/go-log` — `coreerr.E()` error helper
- `forge.lthn.ai/core/go/pkg/core` — `core.Config`, `core.Startable`, `core.ServiceRuntime` interfaces
- `github.com/spf13/viper` — configuration engine
- `github.com/stretchr/testify` — test assertions
