# go-config

`dappco.re/go/config` provides layered configuration loading for Core-based Go
services. It combines file-backed settings, environment variables, and explicit
runtime overrides behind a small `Result`-returning API that matches
`dappco.re/go`.

The package keeps two Viper instances internally. One represents the complete
read view and the other represents only values that should be persisted. This
lets `Commit` write file-backed and explicitly set values without copying
environment secrets into `config.yaml`.

## Install

```bash
go get dappco.re/go/config
```

The module requires Go 1.26 or newer.

## Quick Start

```go
package main

import (
	config "dappco.re/go/config"
	core "dappco.re/go"
)

func main() {
	r := config.New()
	if !r.OK {
		panic(r.Error())
	}
	cfg := r.Value.(*config.Config)

	if set := cfg.Set("dev.editor", "vim"); !set.OK {
		panic(set.Error())
	}
	if commit := cfg.Commit(); !commit.OK {
		panic(commit.Error())
	}

	var editor string
	if get := cfg.Get("dev.editor", &editor); get.OK {
		core.Println(editor)
	}
}
```

## Configuration Sources

Values are resolved in this order:

1. File values loaded from the configured path.
2. Environment variables using `CORE_CONFIG_` by default.
3. Explicit values written with `Set`.

Keys use dot notation, so `CORE_CONFIG_DEV_EDITOR=nano` resolves as
`dev.editor`. `WithEnvPrefix("MYAPP")` changes the environment prefix to
`MYAPP_`.

## Service Mode

`NewConfigService` adapts the package to the Core service lifecycle. Register
it with `core.WithService(config.NewConfigService)`, then call
`ServiceFor[*config.Service]` when a service needs configuration access.

## Development

The compliance gate for this repository is the v0.9.0 audit script from
`core/go`. Local verification should run:

```bash
GOWORK=off go mod tidy
GOWORK=off go vet ./...
GOWORK=off go test -count=1 ./...
gofmt -l .
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```

## Licence

EUPL-1.2
