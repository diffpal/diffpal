# DiffPal Quickstart

This guide is the fastest GitHub path for your first successful DiffPal review.
It uses the Codex API-key recipe because it is a ready-made GitHub Actions
setup, not because Codex is the product boundary.

If you already have another ACP-compatible CLI, start with
[Using Another ACP CLI](ci-examples.md#using-another-acp-cli) instead.

## What You Should See

After the first successful run, expect:

- a `DiffPal Review Summary` PR review
- inline review comments when actionable findings exist
- `.artifacts/diffpal/findings.json` in the workflow workspace
- a failed job only when `gate: true` and blocking findings exist, or when
  setup, authentication, diff collection, or publishing fails

For a fuller outcome checklist, see [what success looks like](what-success-looks-like.md).

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
- a GitHub platform block

The command keeps existing files unless you pass `--force`.

Other setup recipes:

| Setup | Use when |
| --- | --- |
| `codex-api-key` | CI authenticates Codex with `OPENAI_API_KEY`. |
| `codex-subscription` | CI restores local Codex subscription auth. |
| `copilot-github-token` | CI authenticates Copilot with a fine-grained GitHub token. |
| `generic-acp` | You already have another ACP-compatible CLI. |
| `opencode-acp` | You want CI to run OpenCode through ACP. |

For manual setup, copy a config from [`examples/configs`](../examples/configs).

## 2. Add Secret

Add this GitHub repository secret:

| Secret | Purpose |
| --- | --- |
| `OPENAI_API_KEY` | Lets the Codex CLI act as the review provider. |

GitHub provides `GITHUB_TOKEN` automatically. The workflow grants it the
permissions DiffPal needs to publish PR feedback.

For public repositories, do not expose provider credentials to fork PR code.
Keep secret-backed DiffPal review limited to same-repository pull requests and
let forks run no-secret CI only. See [troubleshooting](troubleshooting.md#fork-pull-requests-and-secrets)
for the security rationale.

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

## 4. Open A Same-Repository Pull Request

Open a pull request from a branch in the same repository so the provider secret
is available to the review job. DiffPal should publish the summary, inline
findings, and artifacts described above.

After the first successful run, pin `diffpal-version`, provider CLIs, and bridge
packages when you need fully reproducible credentialed CI.

## Bring Your Own Agent

For another ACP CLI, keep the same workflow shape and replace the provider
install/authentication step plus `.config/diffpal/config.yaml`. Start from the
[generic ACP config](../examples/configs/generic-acp/config.yaml) and the
[CI setup guide](ci-examples.md#using-another-acp-cli).

## Other Hosts And Recipes

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
