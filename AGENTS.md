# Agent Guide

This repository is the `dappco.re/go/config` consumer module for Core
configuration. Work here should follow the current `dappco.re/go` conventions:
public operations return `core.Result`, tests use the Core assertion helpers,
and examples print through `core.Println`.

## Repository Shape

- `config.go` contains the `Config` type, functional options, file loading,
  typed retrieval, persistence, and legacy `Load` / `Save` helpers.
- `env.go` contains environment scanning helpers. It maps prefixed variables to
  lower-case dot keys without importing banned stdlib packages directly.
- `service.go` wraps `Config` as a Core lifecycle service.
- Each production file has a matching `_test.go` and `_example_test.go` sibling.
  Keep tests next to their source file; do not create aggregate compliance
  files.
- `docs/` contains the human-facing architecture and development notes.

## Compliance Rules

The audit script is the work provider:

```bash
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```

Do not stop on partial progress. A compliant run has every counter at `0` and
prints `verdict: COMPLIANT`.

The important local patterns are:

- Use `core.Result` with `core.Ok`, `core.Fail`, or `core.ResultOf`.
- Do not import banned stdlib packages such as `fmt`, `os`, `strings`,
  `path/filepath`, or `encoding/json` in this consumer module.
- Use `core.Path*`, `core.Environ`, `core.SplitN`, `core.Replace`, and
  `core.Sprintf` instead of direct stdlib helpers.
- Public symbols require `Good`, `Bad`, and `Ugly` tests in the matching source
  test file.
- Public symbols require examples in the matching source example file.

## Editing Notes

Keep the dual-view config invariant intact. File-backed and explicitly set
values belong in `Config.file`; the read view in `Config.full` is rebuilt from
file settings, environment variables, and explicit overrides. That separation is
what prevents environment-derived secrets from being written during `Commit`.

Do not edit `BRIEF.md`, `.git`, `.codex`, or any `third_party` directory.
