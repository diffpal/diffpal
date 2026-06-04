# AGENTS

DiffPal development conventions:

## Issue Tracking

This project uses **bd (beads)** for issue tracking.
Run `bd prime` for workflow context, or install hooks (`bd hooks install`) for auto-injection.

Quick reference:

- `bd ready` - Find unblocked work
- `bd create "Title" --type task --priority 2` - Create issue
- `bd close <id>` - Complete work
- `bd dolt push` - Push beads to remote

For full workflow details: `bd prime`

## Repository Conventions

- Use Beads (`bd`) as the source of task truth.
- Keep `dp-` issue IDs in planning comments and commit messages.
- Preserve the repository convention:
  - `cmd/` for executable entrypoints.
  - `internal/` for local implementation packages.
  - `pkg/` for reusable exported packages.
  - `docs/` for decisions and contracts.
  - `.github/` for CI and release workflows.
- Prefer structured JSON outputs for machine interfaces.
- Preserve deterministic IDs and stable output ordering where possible.
- Keep `release` artifacts and local outputs under `.artifacts/` or `dist/`.
- Do not add or preserve tests for removed, legacy, obsolete, or intentionally unsupported behavior.
- Do not add code or tests whose primary purpose is rejecting legacy config or legacy command shapes unless explicitly requested.
