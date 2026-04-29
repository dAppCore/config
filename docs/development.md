---
title: Development
description: Build, test, and compliance workflow for dappco.re/go/config.
---

# Development

This module is a consumer of `dappco.re/go`. Its local style is defined by the
v0.9.0 compliance audit in the Core repository.

## Required Checks

Run these from the repository root:

```bash
GOWORK=off go mod tidy
GOWORK=off go vet ./...
GOWORK=off go test -count=1 ./...
gofmt -l .
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```

The audit script is authoritative. A change is incomplete until the audit
prints `verdict: COMPLIANT` and every counter is `0`.

## Test Layout

Tests are file-aware. A public symbol in `config.go` belongs in
`config_test.go`, a public symbol in `env.go` belongs in `env_test.go`, and a
public symbol in `service.go` belongs in `service_test.go`.

Each public function or method has three variants:

- `Good` for the normal success path.
- `Bad` for expected failure.
- `Ugly` for edge cases such as nil receivers, empty inputs, or boundary
  behaviour.

The canonical name is `Test<File>_<Symbol>_<Variant>`. Do not add aggregate
test files, versioned test files, or AX-7 dump files.

## Examples

Each source file has a matching `_example_test.go` file. Examples must execute
the symbol they document and print stable output through `core.Println`.
Examples should avoid dynamic paths or map iteration unless the output is
normalised first.

## Core Wrapper Policy

Consumer code and tests should not import banned stdlib packages directly.
Common replacements are:

- `core.Sprintf`, `core.Println`, and `core.Print` for formatting.
- `core.PathExt`, `core.PathBase`, `core.PathDir`, and `core.Path` for paths.
- `core.Environ`, `core.Setenv`, and `core.Unsetenv` for environment access.
- `core.NewReader`, `core.SplitN`, `core.Replace`, `core.Lower`, and
  `core.TrimPrefix` for text helpers.

## Adding Behaviour

When adding a public symbol:

1. Add its Good, Bad, and Ugly tests to the matching source test file.
2. Add its runnable example to the matching source example file.
3. Keep production return values in `core.Result` shape.
4. Run the required checks before handing the branch back.
