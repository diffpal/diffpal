# DiffPal

DiffPal reviews pull requests from the diff first, then publishes clear,
policy-aware feedback back to your CI system.

It is built for teams that want AI review output that is easy to scan:

- a PR summary with reviewed files and pass/fail status
- inline comments only for actionable findings
- merge gating through checks/statuses, not bot approvals
- one config file that works across GitHub, GitLab, and Azure DevOps

## Quick Start

Install the CLI and Copilot provider:

```bash
npm install --global @diffpal/diffpal@latest @github/copilot@latest
diffpal init
diffpal doctor
```

Then add DiffPal to your CI:

- [GitHub Actions setup](docs/ci-examples.md#github-actions)
- [GitLab CI setup](docs/ci-examples.md#gitlab-ci)
- [Azure Pipelines setup](docs/ci-examples.md#azure-pipelines)

For production, pin `@diffpal/diffpal` and `@github/copilot` to tested SemVer
versions instead of using `@latest`.

## What DiffPal Publishes

On pull requests, DiffPal can publish:

- a review summary comment
- a required check/status for merge gating
- inline comments or threads for actionable findings
- JSON, SARIF, and CI artifacts for later inspection

The default review checks are:

- `bugs`
- `performance`
- `best-practices`

The default review language is English. Both are configurable in
`.config/diffpal/config.yaml` or by CLI flags.

## Minimal Config

`diffpal init` writes `.config/diffpal/config.yaml`. The default public
onboarding provider is Copilot ACP:

```yaml
version: v1

defaults:
  provider: copilot-acp
  policy: default

providers:
  copilot-acp:
    type: copilot_acp
    copilot_acp:
      extra_args:
        - --stdio

policies:
  default:
    block_on: high

review:
  context_lines: 20
  max_files: 200
  language: en
  checks:
    - bugs
    - performance
    - best-practices
```

## Common Commands

```bash
diffpal doctor --mode github
diffpal review local --base origin/main --head HEAD
diffpal review github --base "$BASE_SHA" --head "$HEAD_SHA" --gate
diffpal review gitlab --base "$BASE_SHA" --head "$HEAD_SHA" --gate
diffpal review ado --base "$BASE_SHA" --head "$HEAD_SHA" --gate
```

## Documentation

- [Quickstart](docs/quickstart.md)
- [CI setup guide](docs/ci-examples.md)
- [Config reference](docs/config-reference.md)
- [Findings schema](docs/findings-schema.md)
- [GitLab adapter reference](docs/platform-gitlab.md)
- [Azure adapter reference](docs/platform-azure.md)
- [Release process](docs/release.md)

## Development

Source development in this repository uses the Go toolchain directly:

```bash
go mod download
go test ./...
go run ./cmd/diffpal --help
```

Maintainers track project work in Beads (`bd`). External contributors do not
need Beads to open issues or pull requests.
