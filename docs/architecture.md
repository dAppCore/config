---
title: Architecture
description: Internal design of dappco.re/go/config.
---

# Architecture

The package is built around one invariant: the configuration a service reads is
not the same thing as the configuration that should be persisted. Runtime
environment values may override file values, but they must never be written back
to disk by `Commit`.

## Config

`Config` owns two Viper instances and a small amount of Core-specific state:

```go
type Config struct {
    mu        sync.RWMutex
    full      *viper.Viper
    file      *viper.Viper
    medium    Medium
    path      string
    envPrefix string
    overrides map[string]any
}
```

`file` stores configuration loaded from files plus explicit `Set` calls.
`full` is rebuilt from `file`, the current environment, and explicit overrides.
Reads use `full`; persistence uses `file`.

## Source Priority

The effective read view is assembled in this order:

1. File-backed settings.
2. Environment variables for the configured prefix.
3. Explicit runtime overrides from `Set`.

The override map exists because explicit `Set` values are persisted in `file`
but must still outrank environment variables in the read view.

## Storage

All file I/O goes through the local `Medium` interface:

```go
type Medium interface {
    Exists(path string) bool
    Read(path string) core.Result
    Write(path, content string) core.Result
    EnsureDir(path string) core.Result
}
```

The default implementation is `(&core.Fs{}).New("/")`. Tests pass a rooted
`core.Fs` created inside `t.TempDir`, which keeps filesystem state deterministic.

## File Loading

`LoadFile` detects the format from the path:

- `.yaml`, `.yml`, and extensionless files load as YAML.
- `.json` loads as JSON.
- `.toml` loads as TOML.
- `.env` loads as dotenv.

Parsed settings merge into `file`, then `full` is rebuilt. A later `Commit`
therefore persists loaded file data and explicit runtime writes, not environment
snapshots.

## Service Wrapper

`Service` embeds `core.ServiceRuntime[ServiceOptions]` and initialises `Config`
during `OnStartup`. Its `Get`, `Set`, `Commit`, and `LoadFile` methods return
`core.Result` and delegate to the loaded `Config`.

`NewConfigService` is the Core service factory. It returns a `core.Result`
containing `*Service`, so it can be passed to `core.WithService`.

## Concurrency

Public `Config` methods lock around Viper access. `Get` and `All` refresh the
read view so environment changes made before the call are visible. `All` returns
a snapshot iterator; later `Set` calls do not mutate an iterator already
returned to the caller.
