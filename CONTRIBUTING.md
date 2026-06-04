# Contributing

## Local flow

1. Fork the repository or create a feature branch.
2. Keep changes scoped to one behavior or contract.
3. Run the local verification commands before opening a PR.
4. Include tests for command behavior, config behavior, and artifact output when applicable.

## Verification

```bash
go mod download
go mod verify
go test ./...
go tool golangci-lint run ./...
go run ./cmd/diffpal --help
```

For Azure DevOps task changes, also run:

```bash
npm --prefix tasks/azure-devops ci
npm --prefix tasks/azure-devops run build
```

## Project conventions

- `cmd/` contains executable entrypoints.
- `internal/` contains DiffPal implementation packages. These are not a supported public Go API.
- `docs/` contains product contracts, config references, and platform behavior.
- Machine-facing output should be structured JSON where practical.
- Keep release artifacts and local outputs under `.artifacts/` or `dist/`.

## Maintainer workflow

- Track tasks in Beads (`bd`).
- Use `bd prime` when starting work to recover workflow context.
- Keep task transitions in one issue state change: `open` -> `in_progress` -> `closed`.
- New features should include command-level acceptance criteria and tests where applicable.

External contributors are not required to use Beads. Maintainers should link accepted work to `dp-` issue IDs when landing changes.

## Beads commands

- `bd prime` - current workflow context and guidance
- `bd status` - overall health
- `bd ready` - unblockable items
- `bd graph` - dependency chains
