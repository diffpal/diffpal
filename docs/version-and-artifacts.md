# Versioning and Artifact Naming

## Status

This document reflects the v1 implementation contract. Release automation should keep
major/minor tags and short SHA tags aligned.

## Version contract

Version data is emitted from `internal/version`.
The build entrypoint is `cmd/diffpal/main.go`, and the root Cobra command tree is rooted in `internal/cmd`.

- `Version`: release or `0.0.0-dev`
- `GitCommit`: short or full VCS revision
- `BuildDate`: RFC3339 timestamp

Build tooling should inject variables with:

```bash
go build -ldflags "-X github.com/diffpal/diffpal/internal/version.Version=v1.2.3 -X github.com/diffpal/diffpal/internal/version.GitCommit=$GIT_SHA -X github.com/diffpal/diffpal/internal/version.BuildDate=$BUILD_DATE" -o dist/diffpal ./cmd/diffpal
```

omnidist is the release path for packaged CLI distributions. Its config should
point `tool.main` at `./cmd/diffpal` and publish the npm package as
`@diffpal/diffpal`.

SemVer applies to CLI, Go module, and Action major compatibility tags.

## Package boundaries

- `cmd/diffpal` is the only executable entrypoint.
- `internal/*` packages are private implementation details of the DiffPal CLI/runtime.
- DiffPal is distributed as a CLI; Go packages under `internal/*` are private implementation details.

## Artifact naming

- `dist/diffpal` — built CLI binary
- `.artifacts/diffpal/findings.json` — canonical bundle
- `.artifacts/diffpal/summary.md` — markdown summary
- `.artifacts/diffpal/diffpal.sarif` — SARIF output
- `@diffpal/diffpal` — npm-distributed CLI package
- `diffpal/action` — GitHub Action wrapper in the separate action repository

GitHub Action release metadata is maintained in the separate `diffpal/action`
repository. Azure DevOps VSIX artifacts are named `dist/diffpal.vsix` and
`dist/diffpal-dev.vsix` in the separate `diffpal/azure-devops` repository.

## Tagging policy

- Action consumers track `v1`.
- npm consumers pin `@diffpal/diffpal` by SemVer or use `latest`.
- Release notes and changelog should reference `go.mod`, `internal/version`, and
  generated artifact checksums.
