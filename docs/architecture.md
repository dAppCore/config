---
title: Architecture
description: Internal design of config -- dual-viper layering, the Medium abstraction, and the framework service wrapper.
---

# Architecture

## Design Goals

1. **Layered resolution** -- a single `Get()` call checks environment, file, and defaults without the caller needing to know which source won.
2. **Safe persistence** -- environment variables must never bleed into the saved config file.
3. **Pluggable storage** -- the file system is abstracted behind `io.Medium`, making tests deterministic and enabling future remote backends.
4. **Framework integration** -- the package satisfies the `core.Config` interface, so any Core service can consume configuration without importing this package directly.

## Key Types

### Config

```go
type Config struct {
    mu     sync.RWMutex
    full   *viper.Viper   // full configuration (file + env + defaults)
    file   *viper.Viper   // file-backed configuration only (for persistence)
    medium coreio.Medium
    path   string
}
```

`Config` is the central type. It holds **two** Viper instances:

- **`full`** -- contains everything: file values, environment bindings, and explicit `Set()` calls. All reads go through `full`.
- **`file`** -- contains only values that originated from the config file or were explicitly set via `Set()`. All writes (`Commit()`) go through `file`.

This dual-instance design is the key architectural decision. It solves the environment leakage problem: Viper merges environment variables into its settings map, which means a naive `SaveConfig()` would serialise env vars into the YAML file. By maintaining `file` as a clean copy, `Commit()` only persists what should be persisted.

### Option

```go
type Option func(*Config)
```

Functional options configure the `Config` at creation time:

| Option            | Effect                                             |
|-------------------|----------------------------------------------------|
| `WithMedium(m)`   | Sets the storage backend (defaults to `io.Local`)  |
| `WithPath(path)`  | Sets the config file path (defaults to `~/.core/config.yaml`) |
| `WithEnvPrefix(p)`| Changes the environment variable prefix (defaults to `CORE_CONFIG`) |

### Service

```go
type Service struct {
    *core.ServiceRuntime[ServiceOptions]
    config *Config
}
```

`Service` wraps `Config` as a framework-managed service. It embeds `ServiceRuntime` for typed options and implements two interfaces:

- **`core.Config`** -- `Get(key, out)` and `Set(key, v)`
- **`core.Startable`** -- `OnStartup(ctx)` triggers config file loading during the application lifecycle

The service is registered as a factory function:

```go
core.New(core.WithService(config.NewConfigService))
```

The factory receives the `*core.Core` instance, constructs the service, and returns it. The framework calls `OnStartup` at the appropriate lifecycle phase, at which point the config file is loaded.

### Env / LoadEnv

```go
func Env(prefix string) iter.Seq2[string, any]
func LoadEnv(prefix string) map[string]any  // deprecated
```

`Env` returns a Go 1.23+ iterator over environment variables matching a given prefix, yielding `(dotKey, value)` pairs. The conversion logic:

1. Filter `os.Environ()` for entries starting with `prefix`
2. Strip the prefix
3. Lowercase and replace `_` with `.`

`LoadEnv` is the older materialising variant and is deprecated in favour of the iterator.

## Data Flow

### Initialisation (`New`)

```
New(opts...)
  |
  +-- create two viper instances (full, file)
  +-- set env prefix on full ("CORE_CONFIG_" default)
  +-- set env key replacer ("." <-> "_")
  +-- apply functional options
  +-- default medium to io.Local if nil
  +-- default path to ~/.core/config.yaml if empty
  +-- enable full.AutomaticEnv()
  +-- if config file exists:
        LoadFile(medium, path)
          +-- medium.Read(path)
          +-- detect config type from extension
          +-- file.MergeConfig(content)
          +-- full.MergeConfig(content)
```

### Read (`Get`)

```
Get(key, &out)
  |
  +-- RLock
  +-- if key == "":
  |     full.Unmarshal(out)  // full config into a struct
  +-- else:
  |     full.IsSet(key)?
  |       yes -> full.UnmarshalKey(key, out)
  |       no  -> error "key not found"
  +-- RUnlock
```

Because `full` has `AutomaticEnv()` enabled, `full.IsSet(key)` returns true if the key exists in the file **or** as a `CORE_CONFIG_*` environment variable.

### Write (`Set` + `Commit`)

```
Set(key, value)
  |
  +-- Lock
  +-- file.Set(key, value)  // track for persistence
  +-- full.Set(key, value)   // make visible to Get()
  +-- Unlock

Commit()
  |
  +-- Lock
  +-- Save(medium, path, file.AllSettings())
  |     +-- yaml.Marshal(data)
  |     +-- medium.EnsureDir(dir)
  |     +-- medium.Write(path, content)
  +-- Unlock
```

Note that `Commit()` serialises `file.AllSettings()`, not `full.AllSettings()`. This is intentional -- it prevents environment variable values from being written to the config file.

### File Loading (`LoadFile`)

`LoadFile` supports YAML, JSON, TOML, and `.env` files. The config type is inferred from the file extension:

- `.yaml`, `.yml` -- YAML
- `.json` -- JSON
- `.toml` -- TOML
- `.env` (no extension, basename `.env`) -- dotenv format

Content is merged (not replaced) into both `full` and `file` via `MergeConfig`, so multiple files can be layered.

## Concurrency

All public methods on `Config` and `Service` are safe for concurrent use. A `sync.RWMutex` protects the internal state:

- `Get`, `All` take a read lock
- `Set`, `Commit`, `LoadFile` take a write lock

## The Medium Abstraction

File operations go through `io.Medium` (from `dappco.re/go/core/io`), not `os` directly. This means:

- **Tests** use `io.NewMockMedium()` -- an in-memory filesystem with a `Files` map
- **Production** uses `io.Local` -- the real local filesystem
- **Future** backends (S3, embedded assets) can implement `Medium` without changing config code

## Compile-Time Interface Checks

The package includes a compile-time assertion that the `Service` type satisfies `core.Startable`:

```go
var _ core.Startable = (*Service)(nil)  // service.go
```

`core.Config` is a concrete struct in upstream Core (not an interface), so this package provides its own `*Config` implementation alongside the Core one. Framework consumers interact with `*config.Service` directly via `core.ServiceFor[*config.Service]`.

## Licence

EUPL-1.2
