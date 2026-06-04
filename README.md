# DiffPal

DiffPal is a diff-first, policy-aware review assistant.

- Diff collection and review are treated as a first-class artifact (`findings.json`).
- Review runs are mode-scoped: local review stays local, and host review emits only that host's artifacts.
- GitHub is the primary host surface; GitLab and Azure adapters follow the same findings contract.

## Package and runtime contract

- Go module: `github.com/diffpal/diffpal`
- CLI binary: `diffpal`
- Default package license: `MIT`
- Build stack: Go 1.26.4
- CLI package: `@diffpal/diffpal`
- GitHub Action: `diffpal/action`
- NPM scope: `@diffpal/*`
- Self-review: `.github/workflows/diffpal-review.yml` installs `@diffpal/diffpal@latest` and runs this repository's action on same-repository pull requests.

See [docs/product-contract.md](docs/product-contract.md) for full contract details.

## Commands

This repository includes the canonical CLI command tree:

- `diffpal review`
- `diffpal review local`
- `diffpal review github`
- `diffpal review gitlab`
- `diffpal review ado`
- `diffpal sarif`
- `diffpal init`
- `diffpal doctor`
- `diffpal version`

## Repository layout

- `cmd/diffpal` contains the build entrypoint for the CLI binary.
- `internal/cmd` contains the Cobra command tree and command wiring.
- `internal/...` contains non-exported runtime, diff, policy, findings, platform, and reliability packages.
- `docs/` records the product contract, config, platform adapters, and release behavior.

## Documentation

- [Quickstart](docs/quickstart.md)
- [Config and policy reference](docs/config-reference.md)
- [Configuration schema](docs/config-schema.md)
- [Findings schema](docs/findings-schema.md)
- [GitHub/GitLab/Azure CI examples](docs/ci-examples.md)
- [GitLab adapter contract](docs/platform-gitlab.md)
- [Azure adapter contract](docs/platform-azure.md)
- [Release process](docs/release.md)

## First Commands

```bash
diffpal init
diffpal doctor
diffpal review local --base origin/main --head HEAD
diffpal review github --base origin/main --head HEAD --gate
```

## Development

```bash
go mod download
go mod verify
go test ./...
go tool golangci-lint run ./...
go run ./cmd/diffpal --help
```

The npm package is the user-facing CLI distribution. Source development in this repository uses the Go toolchain directly.

Maintainers track project work in Beads (`bd`). External contributors do not need Beads to open issues or pull requests.
