# DiffPal Quickstart

This guide gets DiffPal reviewing GitHub pull requests with the fastest
ready-made recipe: Codex API-key auth. If you already have another
ACP-compatible CLI, use the generic ACP config instead:
[`examples/configs/generic-acp/config.yaml`](../examples/configs/generic-acp/config.yaml).

For other provider recipes or CI systems, use the examples matrix:
[`examples/README.md`](../examples/README.md).

## 1. Add Config

Copy the Codex API-key config, or replace it with the generic ACP config for
your own provider CLI:

```bash
mkdir -p .config/diffpal
cp examples/configs/codex-api-key/config.yaml .config/diffpal/config.yaml
```

Or generate a starter config locally with `diffpal init` and compare it with
[`examples/configs/codex-api-key/config.yaml`](../examples/configs/codex-api-key/config.yaml).

## 2. Add Secret

Add this GitHub repository secret:

| Secret | Purpose |
| --- | --- |
| `OPENAI_API_KEY` | Lets the Codex CLI act as the review provider. |

GitHub provides `GITHUB_TOKEN` automatically. The workflow grants it the
permissions DiffPal needs to publish PR feedback.

## 3. Add Workflow

Copy the GitHub Actions example:

```bash
mkdir -p .github/workflows
cp examples/ci/github-actions/codex-api-key.yml .github/workflows/diffpal.yml
```

The example:

- performs a full checkout with `fetch-depth: 0`
- installs the Codex provider command
- authenticates Codex with `OPENAI_API_KEY`
- uses the DiffPal action, which installs the DiffPal CLI
- runs only on trusted same-repository PRs when secrets are required

For another ACP CLI, keep the same workflow shape and replace the provider
install/authentication step plus `.config/diffpal/config.yaml`.

## What Success Looks Like

After a PR run, expect:

- a `diffpal-checks` check run
- a `DiffPal Review Summary` PR comment
- inline comments only when actionable findings exist
- `.artifacts/diffpal/findings.json` in the workflow workspace
- a failed job only when `gate: true` and blocking findings exist, or when
  setup/publish fails

The summary includes a semantic overview of the PR by default. Hide it with:

```yaml
summary-overview: false
```

If you run multiple DiffPal workflows on the same pull request, give each one a
different `review-channel` and `review-id` so their checks and summary comments
stay separate.

## Other Setups

- GitHub Actions: [`examples/ci/github-actions`](../examples/ci/github-actions)
- GitLab CI: [`examples/ci/gitlab`](../examples/ci/gitlab)
- Azure Pipelines: [`examples/ci/azure-pipelines`](../examples/ci/azure-pipelines)
- Provider configs: [`examples/configs`](../examples/configs)
