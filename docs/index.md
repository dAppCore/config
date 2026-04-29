---
title: go-config
description: Layered configuration management for dappco.re/go services.
---

# go-config

`dappco.re/go/config` is the configuration package for Core-based services. It
loads structured settings from YAML, JSON, TOML, dotenv files, environment
variables, and explicit runtime overrides while presenting a small
`core.Result`-based API.

## Module

```text
dappco.re/go/config
```

The module requires Go 1.26 or newer.

## Main APIs

- `New(opts ...Option) core.Result` constructs a `*Config`.
- `WithMedium`, `WithPath`, and `WithEnvPrefix` customise storage, path, and
  environment prefix.
- `(*Config).Get`, `Set`, `Commit`, `LoadFile`, `All`, and `Path` operate on a
  configuration instance.
- `Env` iterates prefixed environment variables as lower-case dot keys.
- `NewConfigService` exposes the same configuration surface as a Core service.

## Result Shape

All operational APIs return `core.Result`. Callers should branch on `r.OK`:

```go
r := config.New()
if !r.OK {
    panic(r.Error())
}
cfg := r.Value.(*config.Config)
```

This is the same shape used by `dappco.re/go` for filesystem, JSON, process,
and service operations.

## Environment Mapping

Environment variables are mapped by stripping the configured prefix, lowering
the remaining name, and replacing `_` with `.`. With the default prefix,
`CORE_CONFIG_DEV_EDITOR=vim` becomes key `dev.editor`.

Use `WithEnvPrefix("MYAPP")` when an application has its own prefix. The
trailing underscore is optional.

## Persistence

`Set` updates both the persisted view and the read view. `Commit` writes only
the persisted view to disk. Environment variables participate in reads but are
not written back, which avoids leaking deployment secrets into config files.

## Related Pages

- [Architecture](architecture.md)
- [Development](development.md)
