---
title: go-config
description: Layered configuration management for the Core framework with file, environment, and in-memory resolution.
---

# go-config

`forge.lthn.ai/core/go-config` provides layered configuration management for applications built on the Core framework. It resolves values through a priority chain -- defaults, file, environment variables, flags -- so that the same codebase works identically across local development, CI, and production without code changes.

## Module Path

```
forge.lthn.ai/core/go-config
```

Requires **Go 1.26+**.

## Quick Start

### Standalone usage

```go
package main

import (
    "fmt"
    config "forge.lthn.ai/core/go-config"
)

func main() {
    cfg, err := config.New()  // loads ~/.core/config.yaml if it exists
    if err != nil {
        panic(err)
    }

    // Write a value and persist it
    _ = cfg.Set("dev.editor", "vim")
    _ = cfg.Commit()

    // Read it back
    var editor string
    _ = cfg.Get("dev.editor", &editor)
    fmt.Println(editor) // "vim"
}
```

### As a Core framework service

```go
import (
    config "forge.lthn.ai/core/go-config"
    "forge.lthn.ai/core/go/pkg/core"
)

app, _ := core.New(
    core.WithService(config.NewConfigService),
)
// The config service loads automatically during OnStartup.
// Retrieve it later via core.ServiceFor[*config.Service](app).
```

## Package Layout

| File            | Purpose                                                        |
|-----------------|----------------------------------------------------------------|
| `config.go`     | Core `Config` struct -- layered Get/Set, file load, commit     |
| `env.go`        | Environment variable iteration and prefix-based loading        |
| `service.go`    | Framework service wrapper with lifecycle (`Startable`) support |
| `config_test.go`| Tests following the `_Good` / `_Bad` / `_Ugly` convention     |

## Dependencies

| Module                            | Role                                    |
|-----------------------------------|-----------------------------------------|
| `forge.lthn.ai/core/go`          | Core framework (`core.Config` interface, `ServiceRuntime`) |
| `forge.lthn.ai/core/go-io`       | Storage abstraction (`Medium` for reading/writing files)    |
| `forge.lthn.ai/core/go-log`      | Contextual error helper (`E()`)         |
| `github.com/spf13/viper`         | Underlying configuration engine         |
| `gopkg.in/yaml.v3`               | YAML serialisation for `Commit()`       |

## Configuration Priority

Values are resolved in ascending priority order:

1. **Defaults** -- hardcoded fallbacks (via `Set()` before any file load)
2. **File** -- YAML loaded from `~/.core/config.yaml` (or a custom path)
3. **Environment variables** -- prefixed with `CORE_CONFIG_` by default
4. **Explicit Set()** -- in-memory overrides applied at runtime

Environment variables always override file values. An explicit `Set()` call overrides everything.

## Key Access

All keys use **dot notation** for nested values:

```go
cfg.Set("a.b.c", "deep")

var val string
cfg.Get("a.b.c", &val) // "deep"
```

This maps to YAML structure:

```yaml
a:
  b:
    c: deep
```

## Environment Variable Mapping

Environment variables are mapped to dot-notation keys by:

1. Stripping the prefix (default `CORE_CONFIG_`)
2. Lowercasing
3. Replacing `_` with `.`

For example, `CORE_CONFIG_DEV_EDITOR=nano` resolves to key `dev.editor` with value `"nano"`.

You can change the prefix with `WithEnvPrefix`:

```go
cfg, _ := config.New(config.WithEnvPrefix("MYAPP"))
// MYAPP_SETTING=secret -> key "setting"
```

## Persisting Changes

`Set()` only writes to memory. Call `Commit()` to flush changes to disk:

```go
cfg.Set("dev.editor", "vim")
cfg.Commit() // writes to ~/.core/config.yaml
```

`Commit()` only persists values that were loaded from the file or explicitly set via `Set()`. Environment variable values are never leaked into the config file.

## Licence

EUPL-1.2
