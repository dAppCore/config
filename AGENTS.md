# Agent Notes

This repository is the `dappco.re/go/config` module. It provides layered
configuration, manifest loading, discovery, feature flags, XDG paths, and the
Core framework service adapter used by other Core projects.

When changing code here, keep the public API aligned with `dappco.re/go` v0.9
patterns. Use the Core wrappers for formatting, JSON, filesystem paths,
environment reads, and assertions in tests. Do not add direct imports of the
stdlib packages banned by the upgrade audit, and do not create compatibility
packages that shadow those stdlib names.

Tests are source-file aware. Public symbols in `config.go` are tested in
`config_test.go`, public symbols in `resolve.go` are tested in
`resolve_test.go`, and so on. Each public symbol needs the Good, Bad, and Ugly
triplet in that sibling file plus a runnable example in the matching
`*_example_test.go` file. Supplemental tests may exist, but their names should
describe the behaviour they cover rather than pretending to be another
symbol's canonical triplet.

The normal verification gate for this repository is:

```bash
GOWORK=off go mod tidy
GOWORK=off go vet ./...
GOWORK=off go test -count=1 ./...
gofmt -l .
bash /Users/snider/Code/core/go/tests/cli/v090-upgrade/audit.sh .
```

`BRIEF.md` is a local work brief and should be left untracked. Do not edit
`third_party/`, `.git/`, or `.codex/` while applying compliance changes.
