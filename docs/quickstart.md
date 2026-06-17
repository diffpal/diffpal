# DiffPal Quickstart

DiffPal is provider-agnostic AI review for pull requests. This guide uses the
default Codex API-key recipe because it is the fastest ready-made GitHub
Actions setup, not because Codex is the product boundary. If you already have
another ACP-compatible CLI, use the generic ACP config instead:
[`examples/configs/generic-acp/config.yaml`](../examples/configs/generic-acp/config.yaml).

For other provider recipes or CI systems, use the examples matrix:
[`examples/README.md`](../examples/README.md).

## 1. Generate Config

Run the onboarding wizard scaffold:

```bash
npx -y @diffpal/diffpal@latest init --wizard --setup codex-api-key --platform github
```

This creates `.config/diffpal/config.yaml` with:

- Codex ACP as the review provider
- `diffpal.gate.block_on: high`
- the standard review checks
- a visible `profiles.ci` profile
- v2 prompt rollout flags enabled in the `ci` profile
- a GitHub platform block

The command keeps existing files unless you pass `--force`.

Other setup recipes:

| Setup | Use when |
| --- | --- |
| `codex-api-key` | CI authenticates Codex with `OPENAI_API_KEY`. |
| `codex-subscription` | CI restores local Codex subscription auth. |
| `copilot-github-token` | CI authenticates Copilot with a fine-grained GitHub token. |
| `generic-acp` | You already have another ACP-compatible CLI. |

For `codex-subscription`, generate `CODEX_AUTH_JSON_B64` with the command
recipe in [`examples/README.md`](../examples/README.md#generate-codex_auth_json_b64).

If you prefer manual setup, copy
[`examples/configs/codex-api-key/config.yaml`](../examples/configs/codex-api-key/config.yaml)
or another recipe from [`examples/configs`](../examples/configs).

## 2. Add Secret

Add this GitHub repository secret:

| Secret | Purpose |
| --- | --- |
| `OPENAI_API_KEY` | Lets the Codex CLI act as the review provider. |

GitHub provides `GITHUB_TOKEN` automatically. The workflow grants it the
permissions DiffPal needs to publish PR feedback.

For public repositories, do not expose provider credentials to fork PR code.
GitHub's fork workflow approval settings control whether outside contributors'
fork workflows run automatically; they do not make it safe to release provider
secrets to fork code. Keep secret-backed DiffPal review limited to
same-repository pull requests, and let forks run no-secret CI only.

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

`pull_request_target` runs from the default branch of the base repository and is
useful for trusted automation such as labeling or commenting. Do not combine it
with checking out the PR head or running fork code such as package installs,
tests, build scripts, hooks, or provider CLIs.

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

## Planned Wizard Flow

`diffpal init --wizard` is the supported entry point for one-command onboarding.
The first implementation generates config safely. The intended full flow is:

- detect GitHub Actions, GitLab CI, or Azure Pipelines config
- choose a provider setup recipe
- choose or name the review profile
- choose gate behavior
- generate `.config/diffpal/config.yaml`
- optionally generate or patch CI configuration after confirmation
