# GitHub Actions

Use this page to run DiffPal in GitHub Actions. For the shortest first setup,
start with the [GitHub quickstart](../getting-started/github-quickstart.md).

## Supported Outputs

- Pull request review summary.
- File-level review comments on changed lines.
- SARIF upload output when enabled by the workflow.
- CI check result from the `diffpal` workflow.

## Prerequisites

- A GitHub repository with Actions enabled.
- Permission to add repository secrets and workflows.
- A committed DiffPal config at `.config/diffpal/config.yaml`.
- A provider secret such as `OPENAI_API_KEY`.

See [Shared Setup](README.md#shared-setup) and
[Providers](../providers/README.md).

## Required Checkout Behavior

Use a full checkout so DiffPal can compare the pull request base and head:

```yaml
- uses: actions/checkout@v4
  with:
    fetch-depth: 0
```

## Required Token And Minimum Permissions

GitHub provides `GITHUB_TOKEN`. Grant the workflow only the permissions DiffPal
needs to read code and publish PR feedback:

```yaml
permissions:
  contents: read
  pull-requests: write
```

`GITHUB_TOKEN` is the host publishing credential. Keep it separate from provider
credentials such as `OPENAI_API_KEY`.

## Provider Installation And Authentication

Install and authenticate the selected provider before the DiffPal step. Use
[Providers](../providers/README.md) for Codex, Copilot, OpenCode, and custom
ACP-compatible CLI setup.

Provider credentials allow the selected third-party provider to process the
review input. Store them as GitHub secrets and keep the credentialed review job
restricted to trusted pull requests. See
[Secrets and fork PRs](../guides/secrets-and-fork-prs.md).

## Minimal Pipeline

```yaml
name: diffpal

on:
  pull_request:
    types: [opened, synchronize, reopened, ready_for_review]

jobs:
  review:
    if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-node@v4
        with:
          node-version: 22

      - name: Install Codex provider
        run: npm install --global @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3

      - name: Authenticate Codex
        run: printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}

      - name: Review pull request
        uses: diffpal/action@v1
        with:
          profile: ci
          base: ${{ github.event.pull_request.base.sha }}
          head: ${{ github.event.pull_request.head.sha }}
          repo: ${{ github.repository }}
          review-id: github-pr-${{ github.event.pull_request.number }}
          feedback: review
          gate: true
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## Feedback Modes

Use `feedback: review` for a PR summary plus file-level comments. Use
`feedback: summary` for the summary and non-file artifacts only.

See [Feedback Modes](README.md#feedback-modes).

## Merge-Gate Setup

Set `gate: true` on `diffpal/action@v1`. Blocking findings fail the workflow
when they meet `diffpal.gate.block_on`.

See [Merge Gates](README.md#merge-gates).

## Fork Or Untrusted-Contribution Behavior

Keep provider credentials out of fork PR code. The minimal pipeline restricts
secret-backed review to same-repository PRs with:

```yaml
if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
```

See [Secrets and fork PRs](../guides/secrets-and-fork-prs.md).

## Expected Results

- A PR review headed `DiffPal Review Summary`.
- Inline review comments when actionable findings exist and feedback is
  `review`.
- `.artifacts/diffpal/findings.json` in the workflow workspace.
- A failed workflow only for blocking gated findings or incomplete review setup.

## Common Failures

- `pull-requests: write` is missing.
- `fetch-depth: 0` is missing.
- `OPENAI_API_KEY` is missing or invalid.
- The PR is from a fork, so the same-repository guard skipped secret-backed
  review.

See [Common Failures](README.md#common-failures).

## Related Examples

- [Codex API key](../../examples/ci/github-actions/codex-api-key.yml)
- [Codex subscription auth](../../examples/ci/github-actions/codex-subscription.yml)
- [Copilot token](../../examples/ci/github-actions/copilot-github-token.yml)

Next step: use [Verify First Review](../getting-started/verify-first-review.md)
after the first GitHub Actions run completes.
