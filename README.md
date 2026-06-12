# DiffPal

DiffPal reviews pull requests from the diff first, then publishes clear,
policy-aware feedback back to your CI system.

It is built for teams that want AI review output that is easy to scan:

- a PR summary with reviewed files and pass/fail status
- inline comments only for actionable findings
- merge gating through checks/statuses, not bot approvals
- one config file that works across GitHub, GitLab, and Azure DevOps

## Quick Start

Add DiffPal to a GitHub pull request workflow. The action installs the DiffPal
CLI; the only provider command you install explicitly here is Copilot.

Create `.github/workflows/diffpal-review.yml`:

```yaml
name: diffpal-review

on:
  pull_request:
    types: [opened, synchronize, reopened, ready_for_review]

concurrency:
  group: diffpal-review-${{ github.event.pull_request.number }}
  cancel-in-progress: true

jobs:
  review:
    if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
      checks: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-node@v4
        with:
          node-version: 22

      - name: Install Copilot provider
        run: npm install --global @github/copilot@latest

      - name: Review pull request
        uses: diffpal/diffpal@v0.1.2
        with:
          diffpal-version: latest
          base: ${{ github.event.pull_request.base.sha }}
          head: ${{ github.event.pull_request.head.sha }}
          repo: ${{ github.repository }}
          review-id: github-pr-${{ github.event.pull_request.number }}
          feedback: balanced
          gate: true
        env:
          COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

Then add `COPILOT_GITHUB_TOKEN` as a repository secret. GitHub provides
`GITHUB_TOKEN` automatically.

For production, pin `diffpal-version` and `@github/copilot` to versions you
have tested.

For other CI systems:

- [GitHub Actions setup](docs/ci-examples.md#github-actions)
- [GitLab CI setup](docs/ci-examples.md#gitlab-ci)
- [Azure Pipelines setup](docs/ci-examples.md#azure-pipelines)

## What You Should See

On the pull request, DiffPal publishes:

- a `diffpal-checks` check run
- a `DiffPal Review Summary` comment with a semantic summary of the change
- inline comments only when there are actionable findings
- a failed job only when `gate: true` and blocking findings exist, or when
  setup fails

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

Commit `.config/diffpal/config.yaml` to choose the provider and review policy.
The default public onboarding provider is Copilot ACP:

```yaml
version: v1

defaults:
  provider: copilot-acp
  policy: default

providers:
  copilot-acp:
    type: copilot_acp
    copilot_acp:
      model: gpt-5-mini

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

You can generate a starting config locally with `diffpal init`, then commit the
file after reviewing it.

## Common Commands

```bash
diffpal init
diffpal doctor --mode github
diffpal review local --base origin/main --head HEAD
diffpal review github --base "$BASE_SHA" --head "$HEAD_SHA" --feedback balanced --gate
diffpal review gitlab --base "$BASE_SHA" --head "$HEAD_SHA" --feedback balanced --gate
diffpal review ado --base "$BASE_SHA" --head "$HEAD_SHA" --feedback balanced --gate
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
