<!-- SPDX-License-Identifier: EUPL-1.2 -->

# config

> Config primitives — schemas, conclave, env, watch, resolve, workspace

[![CI](https://github.com/dappcore/config/actions/workflows/ci.yml/badge.svg?branch=dev)](https://github.com/dappcore/config/actions/workflows/ci.yml)
[![Quality Gate](https://sonarcloud.io/api/project_badges/measure?project=dappcore_config&metric=alert_status)](https://sonarcloud.io/dashboard?id=dappcore_config)
[![Coverage](https://codecov.io/gh/dappcore/config/branch/dev/graph/badge.svg)](https://codecov.io/gh/dappcore/config)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_config&metric=security_rating)](https://sonarcloud.io/dashboard?id=dappcore_config)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_config&metric=sqale_rating)](https://sonarcloud.io/dashboard?id=dappcore_config)
[![Reliability Rating](https://sonarcloud.io/api/project_badges/measure?project=dappcore_config&metric=reliability_rating)](https://sonarcloud.io/dashboard?id=dappcore_config)
[![Code Smells](https://sonarcloud.io/api/project_badges/measure?project=dappcore_config&metric=code_smells)](https://sonarcloud.io/dashboard?id=dappcore_config)
[![Lines of Code](https://sonarcloud.io/api/project_badges/measure?project=dappcore_config&metric=ncloc)](https://sonarcloud.io/dashboard?id=dappcore_config)
[![Go Reference](https://pkg.go.dev/badge/dappco.re/go/config.svg)](https://pkg.go.dev/dappco.re/go/config)
[![License: EUPL-1.2](https://img.shields.io/badge/License-EUPL--1.2-blue.svg)](https://eupl.eu/1.2/en/)


`dappco.re/go/config` is the Core configuration module. It gives Core services
and command-line tools a single way to resolve configuration from defaults,
project `.core/` files, user-global files, environment variables, and explicit
runtime writes.

The package is built around a `Config` type that keeps two views of state. The
read view includes file data, defaults, environment values, and in-memory
updates. The write view contains only file-backed and explicit values, so
`Commit` can persist configuration without leaking environment overrides into
the YAML file.

## Main Capabilities

- Load and save YAML configuration through the `dappco.re/go/io` medium
  abstraction.
- Discover `.core/config.yaml` files by walking up from a project directory.
- Resolve typed Core manifests such as `build.yaml`, `test.yaml`,
  `workspace.yaml`, `manifest.yaml`, `agent.yaml`, and `zone.yaml`.
- Sign and verify view and package manifests with ed25519 signatures.
- Expose feature flags through config, process-level defaults, and environment
  overrides.
- Run as a Core service with lifecycle startup, actions, commands, and optional
  filesystem watching.

## Basic Usage

```go
package main

import (
    core "dappco.re/go"
    config "dappco.re/go/config"
    coreio "dappco.re/go/io"
)

func main() {
    cfg, err := config.New(
        config.WithMedium(coreio.Local),
        config.WithPath(".core/config.yaml"),
    )
    if err != nil {
        panic(err)
    }

    _ = cfg.Set("dev.editor", "vim")
    _ = cfg.Commit()

    var editor string
    _ = cfg.Get("dev.editor", &editor)
    core.Println(editor)
}
```

## Development

This module follows the Core v0.9 compliance shape. Keep tests next to the
source file they cover, keep examples in sibling `*_example_test.go` files, and
use Core assertion and wrapper helpers throughout tests. The repository audit
at `/Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh` is the final
contract for compliance.

See `docs/index.md`, `docs/architecture.md`, and `docs/development.md` for the
longer API, design, and contribution notes.
