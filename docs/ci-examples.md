# CI Setup Guide

This guide explains how DiffPal behaves in CI. Copy-paste files live in
[`examples/`](../examples/README.md).

## Common Setup

DiffPal's portability point is provider install/auth: CI chooses and
authenticates the provider, while DiffPal keeps the PR review workflow,
artifacts, and publishing behavior consistent across hosts.

Every CI system needs:

1. A full git checkout, so DiffPal can compare base and head commits.
2. The provider CLI runtime required by your selected agent. The maintained
   examples use Node.js because those provider CLIs are installed with npm.
3. A DiffPal config committed at `.config/diffpal/config.yaml`.
4. A provider auth secret.
5. A platform token so DiffPal can publish PR feedback.

Choose a ready-made provider recipe or configure `generic_acp` for your own ACP
CLI. The selected provider lives under `runtime.providers`; the CI-specific
steps are the install and authentication commands for that provider.

| Setup | Config | Required secret |
| --- | --- | --- |
| Generic ACP CLI | [`examples/configs/generic-acp/config.yaml`](../examples/configs/generic-acp/config.yaml) | provider-specific |
| Codex API key | [`examples/configs/codex-api-key/config.yaml`](../examples/configs/codex-api-key/config.yaml) | `OPENAI_API_KEY` |
| Codex subscription auth | [`examples/configs/codex-subscription/config.yaml`](../examples/configs/codex-subscription/config.yaml) | `CODEX_AUTH_JSON_B64` |
| Copilot token | [`examples/configs/copilot-github-token/config.yaml`](../examples/configs/copilot-github-token/config.yaml) | `COPILOT_GITHUB_TOKEN` |
| OpenCode ACP | [`examples/configs/opencode-acp/config.yaml`](../examples/configs/opencode-acp/config.yaml) | OpenCode-specific |

For Codex subscription auth, generate `CODEX_AUTH_JSON_B64` with the command
recipe in [`examples/README.md`](../examples/README.md#generate-codex_auth_json_b64),
then store it as a protected or masked CI secret.

## Using Another ACP CLI

DiffPal can use any CLI that starts an ACP stdio server. Copy the CI example
for your host, replace the provider install/authentication step with your CLI's
setup, and use a config like:

```yaml
runtime:
  providers:
    my-review-agent:
      type: generic_acp
      generic_acp:
        cmd: ["your-acp-cli", "acp", "--stdio"]

diffpal:
  provider: my-review-agent
```

The rest of the GitHub, GitLab, and Azure examples stay the same: checkout the
full git history, run DiffPal with the selected profile, and pass the platform
token for publishing feedback.

OpenCode is available as a first-class ACP alias:

```yaml
runtime:
  providers:
    opencode-acp:
      type: opencode_acp
      opencode_acp:
        model: opencode/big-pickle

diffpal:
  provider: opencode-acp
```

Install and authenticate `opencode` in CI before the DiffPal step.

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
```

Use a same-repository PR guard before exposing provider secrets:

```yaml
if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
```

GitHub Actions settings checklist for fork PRs:

- Set fork workflow approval to the strictest option you can tolerate, ideally
  approval for all external contributors. This controls whether outside
  contributors' fork workflows run automatically; it does not release provider
  secrets to fork code.
- Do not enable settings that send secrets or write tokens to fork pull request
  workflows.
- Keep explicit minimal `permissions`; start with `contents: read` for no-secret
  PR CI and grant `pull-requests: write` only to jobs that need to publish
  feedback.
- Prefer GitHub-hosted runners for untrusted PRs. Self-hosted runners can be
  persistently affected by untrusted workflow code.
- Pin third-party actions to full commit SHAs where practical, especially in
  secret-bearing jobs.

`pull_request_target` runs from the default branch of the base repository and is
useful for trusted automation such as labeling or commenting. It always runs in
the base repository context, so do not combine it with checking out the PR head
or running fork code such as package installs, tests, build scripts, hooks, or
provider CLIs. Fork PRs should run no-secret CI only; provider-backed DiffPal
review is for same-repository PRs, trusted branches, or maintainer-controlled
automation that does not execute fork code.

What you should see:

- A PR review headed `DiffPal Review Summary`.
- Inline review comments when DiffPal finds actionable issues.
- Job failure only when `gate` is set and blocking findings exist, or when setup/publish fails.

Common fixes:

- `GITHUB_TOKEN is required`: keep `GITHUB_TOKEN` on the review step.
- No PR review: confirm `pull-requests: write`.
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
The examples restrict secret-backed review jobs to same-project merge requests
with `$CI_MERGE_REQUEST_SOURCE_PROJECT_PATH == $CI_PROJECT_PATH`.

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
- Run the task from PR validation or an Azure branch policy. When `base` and
  `head` are omitted, the task fetches the target branch and computes the PR
  merge-base automatically.
- Set `explain: true` while debugging to print the resolved PR id, branches,
  base/head, merge-base, and redacted CLI arguments.
- Keep credentialed steps behind `ne(variables['System.PullRequest.IsFork'], 'True')`
  or a stricter organization-specific trusted-source condition.

What you should see:

- Azure PR threads for actionable findings.
- An Azure PR summary thread headed `DiffPal Review Summary`.
- Azure PR status named `DiffPal Review`.
- Failed task when `gate` is true and blocking findings exist.

Common fixes:

- `SYSTEM_ACCESSTOKEN` is empty: enable OAuth token access for scripts.
- Non-PR build failure: configure the pipeline as PR validation or pass explicit
  `base` and `head` revisions for an advanced manual run.
- Target ref or merge-base failure: keep `fetchDepth: 0` and ensure the target
  branch is fetchable from `origin`.
- Task cannot find `diffpal`: keep `install: true`, or set `install: false`
  only when `diffpal` is already on `PATH`.
- Custom binary path: set `diffpalPath`; custom paths skip automatic install.
- Status does not block merge: add an Azure branch status policy for DiffPal.

## Feedback and Outputs

Use `feedback` for normal setup:

| Feedback | Behavior |
| --- | --- |
| `summary` | PR/MR summary. On GitHub, actionable findings are still published as inline PR review comments. |
| `balanced` | Summary plus actionable high-confidence inline comments or threads. |
| `inline` | Summary plus a more permissive inline threshold for actionable findings. |

Raw `mode` remains available for advanced publish-surface control and overrides
`feedback` when set.

The semantic change overview is shown by default in PR reviews.
Turn it off with `summary-overview: false` in GitHub Actions or
`--summary-overview=false` on the CLI.

For parallel GitHub review channels, set `review-channel`. The default channel
is `diffpal`, which publishes a DiffPal PR review. A dev channel such as
`diffpal-dev` publishes a separate PR review:

```yaml
with:
  review-channel: diffpal-dev
  review-id: github-pr-${{ github.event.pull_request.number }}-diffpal-dev
```

Default balanced publish modes:

| Platform | Default modes |
| --- | --- |
| GitHub | `comments,sarif,summary` |
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
