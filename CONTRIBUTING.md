# Contributing To DiffPal

Thanks for helping improve DiffPal.

## Before You Start

- Keep PRs scoped to one behavior, contract, or documentation improvement.
- If you are fixing setup or publishing behavior, include the exact environment
  and CI host.
- External contributors do not need to use the internal Beads workflow.

## Local Setup

```bash
go mod download
go mod verify
go test ./...
go tool golangci-lint run ./...
go run ./cmd/diffpal --help
```

GitHub Action wrapper changes live in the separate `diffpal/action` repository.
Azure DevOps Marketplace extension changes live in the separate
`diffpal/azure-devops` repository. Each integration repo uses its own Node.js
verification commands.

## Optional Integration Checks

Provider-backed reviewer checks are tagged integration tests. They are not part
of default PR verification because they need local provider auth, network
access, and available provider quota.

```bash
# Codex ACP review path; uses npx -y @normahq/codex-acp-bridge@latest.
go test -tags='integration,codex' -count=1 ./internal/reviewer \
  -run TestADKRuntimeCodexACPReviewFindsUnsafeHandler -v

# Copilot ACP provider error path; uses copilot --acp --stdio.
go test -tags='integration,copilot' -count=1 ./internal/reviewer \
  -run TestADKRuntimeCopilotACPProviderErrorPath -v
```

## Repository Conventions

- `cmd/` contains executable entrypoints.
- `internal/` contains DiffPal implementation packages. These are not a
  supported public Go API.
- `docs/` contains user documentation, config references, and platform behavior.
- Machine-facing output should be structured JSON where practical.
- Keep release artifacts and local outputs under `.artifacts/` or `dist/`.
- Use Conventional Commits when possible.

## Pull Request Checklist

- [ ] One focused change
- [ ] Tests or docs added where applicable
- [ ] Setup, config, or output changes documented
- [ ] CI examples updated if user workflow changed
- [ ] PR description explains the expected user-visible outcome

## Maintainer Workflow

- Track tasks in Beads (`bd`).
- Use `bd prime` when starting work to recover workflow context.
- Keep task transitions in one issue state change: `open` -> `in_progress` -> `closed`.
- Link accepted work to `dp-` issue IDs when landing changes.
- New features should include command-level acceptance criteria and tests where
  applicable.

## Beads Commands

- `bd prime` - current workflow context and guidance
- `bd status` - overall health
- `bd ready` - unblockable items
- `bd graph` - dependency chains
