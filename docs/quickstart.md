# DiffPal Quickstart

This guide gets DiffPal reviewing GitHub pull requests with the Copilot ACP
provider. The GitHub Action installs the DiffPal CLI for you; you only install
the provider command in the workflow.

For GitLab CI and Azure Pipelines, use the full
[CI setup guide](ci-examples.md).

## 1. Add Config

Commit `.config/diffpal/config.yaml`:

```yaml
version: v1

runtime:
  providers:
    copilot-acp:
      type: copilot_acp
      copilot_acp:
        model: gpt-5-mini

diffpal:
  provider: copilot-acp
  gate:
    block_on: high
  review:
    language: en
    checks:
      - bugs
      - performance
      - best-practices
```

You can generate a starter config locally with `diffpal init`, but the important
part is committing the reviewed config file before CI runs.

## 2. Add Secret

Add this repository secret:

| Secret | Purpose |
| --- | --- |
| `COPILOT_GITHUB_TOKEN` | Lets the Copilot CLI act as the review provider. |

GitHub provides `GITHUB_TOKEN` automatically. The workflow below grants it the
permissions DiffPal needs to publish PR feedback.

## 3. Add Workflow

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

## What Success Looks Like

After a PR run, expect:

- a `diffpal-checks` check run
- a `DiffPal Review Summary` PR comment
- inline comments only when actionable findings exist
- `.artifacts/diffpal/findings.json` in the workflow workspace
- a failed job only when `gate: true` and blocking findings exist, or when
  setup fails

The summary includes a semantic overview of the PR by default. Hide it with:

```yaml
summary-overview: false
```

## Local Commands

Local commands are useful for setup checks and debugging, but they are not the
main quickstart path.

```bash
npm install --global @diffpal/diffpal@latest @github/copilot@latest
diffpal init
diffpal doctor --mode github
diffpal review local --base origin/main --head HEAD
```

## Other CI Systems

- [GitLab CI](ci-examples.md#gitlab-ci)
- [Azure Pipelines](ci-examples.md#azure-pipelines)

For production, pin `diffpal-version`, `@diffpal/diffpal`, and
`@github/copilot` to versions you have tested.
