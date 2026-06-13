# CI Setup Guide

This guide explains how DiffPal behaves in CI. Copy-paste files live in
[`examples/`](../examples/README.md).

## Common Setup

Every CI system needs:

1. A full git checkout, so DiffPal can compare base and head commits.
2. Node.js, because the public provider CLIs are installed with npm.
3. A DiffPal config committed at `.config/diffpal/config.yaml`.
4. A provider auth secret.
5. A platform token so DiffPal can publish PR feedback.

Choose one provider setup:

| Setup | Config | Required secret |
| --- | --- | --- |
| Codex API key | [`examples/configs/codex-api-key/config.yaml`](../examples/configs/codex-api-key/config.yaml) | `OPENAI_API_KEY` |
| Codex subscription auth | [`examples/configs/codex-subscription/config.yaml`](../examples/configs/codex-subscription/config.yaml) | `CODEX_AUTH_JSON_B64` |
| Copilot token | [`examples/configs/copilot-github-token/config.yaml`](../examples/configs/copilot-github-token/config.yaml) | `COPILOT_GITHUB_TOKEN` |

## GitHub Actions

Examples:

- [Codex API key](../examples/ci/github-actions/codex-api-key.yml)
- [Codex subscription auth](../examples/ci/github-actions/codex-subscription.yml)
- [Copilot token](../examples/ci/github-actions/copilot-github-token.yml)

Required permissions:

```yaml
permissions:
  contents: read
  pull-requests: write
  checks: write
```

Use a same-repository PR guard before exposing provider secrets:

```yaml
if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
```

What you should see:

- `diffpal-checks` check run on the PR head commit.
- A PR summary comment headed `DiffPal Review Summary`.
- Inline comments when DiffPal finds actionable issues.
- Job failure only when `gate` is set and blocking findings exist, or when setup/publish fails.

Common fixes:

- `GITHUB_TOKEN is required`: keep `GITHUB_TOKEN` on the review step.
- No summary comment: confirm `pull-requests: write`.
- No check run: confirm `checks: write`.
- Fork PRs do not run: this is intentional when using secrets.

## GitLab CI

Examples:

- [Codex API key](../examples/ci/gitlab/codex-api-key.yml)
- [Codex subscription auth](../examples/ci/gitlab/codex-subscription.yml)
- [Copilot token](../examples/ci/gitlab/copilot-github-token.yml)

Required variables:

| Name | Purpose |
| --- | --- |
| `CI_JOB_TOKEN` | Built-in token, when your instance allows MR API publishing. |
| `GITLAB_TOKEN` | Optional dedicated token when `CI_JOB_TOKEN` is not enough. |

Use protected/masked variables for provider tokens. If your project accepts fork
merge requests, keep provider tokens available only to trusted pipelines.

What you should see:

- GitLab discussions for actionable findings.
- Code Quality and SARIF artifacts.
- `.artifacts/diffpal/summary.md` in job artifacts.
- Failed job when `--gate` is set and blocking findings exist.

Common fixes:

- Missing base SHA: run only on merge request pipelines.
- Publish denied: use `GITLAB_TOKEN` instead of `CI_JOB_TOKEN`.
- Diff is incomplete: keep `GIT_DEPTH: "0"`.

## Azure Pipelines

Examples:

- [Codex API key](../examples/ci/azure-pipelines/codex-api-key.yml)
- [Codex subscription auth](../examples/ci/azure-pipelines/codex-subscription.yml)
- [Copilot token](../examples/ci/azure-pipelines/copilot-github-token.yml)

Required setup:

- Enable **Allow scripts to access the OAuth token**.
- Pass `SYSTEM_ACCESSTOKEN: $(System.AccessToken)` to the `DiffPalReview@1` task.
- Keep `fetchDepth: 0` on checkout.

What you should see:

- Azure PR threads for actionable findings.
- An Azure PR summary thread headed `DiffPal Review Summary`.
- Azure PR status named `DiffPal Review`.
- Failed task when `gate` is true and blocking findings exist.

Common fixes:

- `SYSTEM_ACCESSTOKEN` is empty: enable OAuth token access for scripts.
- Task cannot find `diffpal`: keep `install: true`, or set `install: false`
  only when `diffpal` is already on `PATH`.
- Custom binary path: set `diffpalPath`; custom paths skip automatic install.
- Status does not block merge: add an Azure branch status policy for DiffPal.

## Feedback and Outputs

Use `feedback` for normal setup:

| Feedback | Behavior |
| --- | --- |
| `summary` | PR summary plus status/check, no inline comments or threads. |
| `balanced` | Summary plus actionable high-confidence inline comments or threads. |
| `inline` | Summary plus a more permissive inline threshold for actionable findings. |

Raw `mode` remains available for advanced publish-surface control and overrides
`feedback` when set.

The semantic change overview is shown by default in summary comments/checks.
Turn it off with `summary-overview: false` in GitHub Actions or
`--summary-overview=false` on the CLI.

Default balanced publish modes:

| Platform | Default modes |
| --- | --- |
| GitHub | `check-run,comments,sarif,summary` |
| GitLab | `code-quality,discussions,sarif,summary` |
| Azure | `threads,status,summary` |

Common artifacts:

| Path | Purpose |
| --- | --- |
| `.artifacts/diffpal/findings.json` | Canonical structured findings bundle. |
| `.artifacts/diffpal/summary.md` | Human-readable review summary. |
| `.artifacts/diffpal/diffpal.sarif` | SARIF report when enabled by the platform mode. |
| `.artifacts/diffpal/codequality.json` | GitLab Code Quality report. |

## Production Hardening

- Pin npm package versions after the first successful setup.
- Keep provider secrets out of untrusted fork pipelines.
- Start with `block_on: high`; lower the threshold only after tuning policy.
- Keep `fetch-depth: 0`, `GIT_DEPTH: "0"`, or `fetchDepth: 0` in CI.
- Run `diffpal doctor --mode <host>` before enabling a blocking gate.
