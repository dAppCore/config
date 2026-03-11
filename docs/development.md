---
title: Development
description: How to build, test, and contribute to go-config.
---

# Development

## Prerequisites

- **Go 1.26+**
- **Core CLI** (`core` binary) for running tests and quality checks
- The Go workspace at `~/Code/go.work` should include this module

## Running Tests

```bash
cd /path/to/go-config

# All tests
core go test

# Single test
core go test --run TestConfig_Get_Good

# With coverage
core go cov
core go cov --open   # opens HTML report in browser
```

### Test Naming Convention

Tests follow the `_Good` / `_Bad` / `_Ugly` suffix pattern:

| Suffix  | Meaning                         |
|---------|---------------------------------|
| `_Good` | Happy path -- expected success  |
| `_Bad`  | Expected error conditions       |
| `_Ugly` | Panics, edge cases, corruption  |

### Mock Medium

Tests use `io.NewMockMedium()` to avoid touching the real filesystem. Pre-populate it by writing directly to the `Files` map:

```go
m := io.NewMockMedium()
m.Files["/tmp/test/config.yaml"] = "app:\n  name: existing\n"

cfg, err := config.New(config.WithMedium(m), config.WithPath("/tmp/test/config.yaml"))
```

This pattern keeps tests fast, deterministic, and parallelisable.

## Quality Checks

```bash
# Format, vet, lint, test in one pass
core go qa

# Full suite (adds race detector, vulnerability scan, security audit)
core go qa full

# Individual commands
core go fmt
core go vet
core go lint
```

## Code Style

- **UK English** in comments and documentation (colour, organisation, centre)
- **`declare(strict_types=1)`** equivalent: all functions have explicit parameter and return types
- **Error wrapping**: use `coreerr.E(caller, message, underlying)` from `go-log`
- **Formatting**: standard `gofmt` / `goimports`

## Project Structure

```
go-config/
    .core/
        build.yaml       # Build configuration (targets, flags)
        release.yaml     # Release configuration (changelog rules)
    config.go            # Config struct, New(), Get/Set/Commit, Load/Save
    config_test.go       # Tests
    env.go               # Env() iterator, LoadEnv() (deprecated)
    service.go           # Framework service wrapper (Startable)
    go.mod
    go.sum
    docs/
        index.md         # This documentation
        architecture.md  # Internal design
        development.md   # Build and contribution guide
```

## Adding a New Feature

1. **Write the test first** -- add a `TestFeatureName_Good` (and `_Bad` if error paths exist) to `config_test.go`.
2. **Implement** -- keep the dual-viper invariant: writes go to both `v` and `f`; reads come from `v`; persistence comes from `f`.
3. **Run QA** -- `core go qa` must pass before committing.
4. **Update docs** -- if the change affects public API, update `docs/index.md` and `docs/architecture.md`.

## Interface Compliance

`Config` and `Service` both satisfy `core.Config`. `Service` additionally satisfies `core.Startable`. These are enforced at compile time:

```go
var _ core.Config    = (*Config)(nil)
var _ core.Config    = (*Service)(nil)
var _ core.Startable = (*Service)(nil)
```

If you add a new interface method upstream in `core/go`, the compiler will tell you what to implement here.

## Commit Guidelines

- Use conventional commits: `type(scope): description`
- Include `Co-Authored-By: Claude Opus 4.6 <noreply@anthropic.com>` when pair-programming with Claude
- Push via SSH: `ssh://git@forge.lthn.ai:2223/core/go-config.git`

## Licence

EUPL-1.2
