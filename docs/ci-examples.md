# CI Setup Guide

This is the main setup guide for running DiffPal in pull request pipelines.
The default provider path is Copilot ACP.

For quick onboarding, examples use `@latest`. For production, pin
`@diffpal/diffpal` and `@github/copilot` to versions you have tested.

## Common Setup

Every CI system needs:

1. A full git checkout, so DiffPal can compare base and head commits.
2. Node.js, because the public CLI and Copilot provider are installed with npm.
3. A DiffPal config committed at `.config/diffpal/config.yaml`.
4. `COPILOT_GITHUB_TOKEN` as a secret for the Copilot provider.
5. A platform token so DiffPal can publish PR feedback.

Run this locally first:

```bash
npm install --global @diffpal/diffpal@latest @github/copilot@latest
diffpal init
diffpal doctor
```

The generated config defaults to:

```yaml
defaults:
  provider: copilot-acp
  policy: default

providers:
  copilot-acp:
    type: copilot_acp
    copilot_acp:
      model: gpt-5.4-mini

review:
  language: en
  checks:
    - bugs
    - performance
    - best-practices
```

## GitHub Actions

### Required Secrets and Permissions

| Name | Required | Purpose |
| --- | --- | --- |
| `COPILOT_GITHUB_TOKEN` | yes | Authenticates the Copilot CLI provider. |
| `GITHUB_TOKEN` | built in | Publishes check runs, summary comments, and inline comments. |

Workflow permissions:

```yaml
permissions:
  contents: read
  pull-requests: write
  checks: write
```

Use a same-repository PR guard before exposing `COPILOT_GITHUB_TOKEN`.

### Workflow

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
        uses: diffpal/action@v1
        with:
          base: ${{ github.event.pull_request.base.sha }}
          head: ${{ github.event.pull_request.head.sha }}
          repo: ${{ github.repository }}
          review-id: github-pr-${{ github.event.pull_request.number }}
          language: en
          review-checks: bugs,performance,best-practices
          feedback: balanced
          summary-overview: true
          gate: true
        env:
          COPILOT_GITHUB_TOKEN: ${{ secrets.COPILOT_GITHUB_TOKEN }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### What You Should See

- `diffpal-checks` check run on the PR head commit.
- A PR summary comment headed `DiffPal Review Summary`.
- Inline comments when DiffPal finds actionable issues.
- Job failure only when `--gate` is set and blocking findings exist, or when setup/publish fails.

### Common GitHub Fixes

- `GITHUB_TOKEN is required`: keep the `env` block on the review step.
- No summary comment: confirm `pull-requests: write`.
- No check run: confirm `checks: write`.
- To pin the CLI, set `diffpal-version: 0.1.x` on `diffpal/action@v1`.
- To hide the change overview section, set `summary-overview: false`.
- Fork PRs do not run: this is intentional when using secrets.

## GitLab CI

### Required Variables

| Name | Required | Purpose |
| --- | --- | --- |
| `COPILOT_GITHUB_TOKEN` | yes | Authenticates the Copilot CLI provider. |
| `CI_JOB_TOKEN` | usually | Publishes from GitLab CI when allowed by your instance. |
| `GITLAB_TOKEN` | optional | Dedicated API token when `CI_JOB_TOKEN` is not enough. |

Use protected/masked variables for tokens. If your project accepts fork merge
requests, keep provider tokens available only to trusted pipelines.

### Pipeline

Add this to `.gitlab-ci.yml`:

```yaml
stages:
  - review

diffpal-review:
  stage: review
  image: node:22
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
  resource_group: "diffpal:$CI_MERGE_REQUEST_IID"
  before_script:
    - npm install --global @diffpal/diffpal@latest @github/copilot@latest
  script:
    - diffpal doctor --mode gitlab
    - >-
      diffpal review gitlab
      --base "$CI_MERGE_REQUEST_DIFF_BASE_SHA"
      --head "$CI_COMMIT_SHA"
      --repo "$CI_PROJECT_PATH"
      --review-id "gitlab-mr-$CI_MERGE_REQUEST_IID"
      --language en
      --review-checks bugs,performance,best-practices
      --feedback balanced
      --gate
  variables:
    GIT_DEPTH: "0"
  artifacts:
    when: always
    paths:
      - .artifacts/diffpal/
    reports:
      codequality: .artifacts/diffpal/codequality.json
      sarif: .artifacts/diffpal/diffpal.sarif
```

### What You Should See

- GitLab discussions for actionable findings.
- Code Quality and SARIF artifacts.
- `.artifacts/diffpal/summary.md` in job artifacts.
- Failed job when `--gate` is set and blocking findings exist.

### Common GitLab Fixes

- Missing base SHA: run only on merge request pipelines.
- Publish denied: use `GITLAB_TOKEN` instead of `CI_JOB_TOKEN`.
- Diff is incomplete: keep `GIT_DEPTH: "0"`.

## Azure Pipelines

### Required Setup

| Name | Required | Purpose |
| --- | --- | --- |
| `COPILOT_GITHUB_TOKEN` | yes | Authenticates the Copilot CLI provider. |
| `SYSTEM_ACCESSTOKEN` | yes | Publishes Azure PR threads and status. |

In Azure Pipelines, enable **Allow scripts to access the OAuth token** so
`$(System.AccessToken)` is available to the task.

The `DiffPalReview@1` task expects `diffpal` to already be on `PATH`.

### Pipeline

Add this to `azure-pipelines.yml`:

```yaml
trigger: none
pr:
  - main

pool:
  vmImage: ubuntu-latest

steps:
  - checkout: self
    fetchDepth: 0

  - task: NodeTool@0
    inputs:
      versionSpec: "22.x"

  - script: npm install --global @diffpal/diffpal@latest @github/copilot@latest
    displayName: Install DiffPal and Copilot

  - script: diffpal doctor --mode ado
    displayName: Check DiffPal setup
    env:
      COPILOT_GITHUB_TOKEN: $(COPILOT_GITHUB_TOKEN)
      SYSTEM_ACCESSTOKEN: $(System.AccessToken)

  - task: DiffPalReview@1
    displayName: DiffPal review
    inputs:
      language: en
      reviewChecks: bugs,performance,best-practices
      feedback: balanced
      gate: true
    env:
      COPILOT_GITHUB_TOKEN: $(COPILOT_GITHUB_TOKEN)
      SYSTEM_ACCESSTOKEN: $(System.AccessToken)
```

### What You Should See

- Azure PR threads for actionable findings.
- An Azure PR summary thread headed `DiffPal Review Summary`.
- Azure PR status named `DiffPal Review`.
- Failed task when `gate` is true and blocking findings exist.

### Common Azure Fixes

- `SYSTEM_ACCESSTOKEN` is empty: enable OAuth token access for scripts.
- Task cannot find `diffpal`: install the CLI first or set `diffpalPath`.
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

The semantic change overview is shown by default in summary comments/checks. Turn it off
with `summary-overview: false` in GitHub Actions or
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
- Keep `COPILOT_GITHUB_TOKEN` out of untrusted fork pipelines.
- Start with `--block-on high`; lower the threshold only after tuning policy.
- Keep `fetch-depth: 0`, `GIT_DEPTH: "0"`, or `fetchDepth: 0` in CI.
- Run `diffpal doctor --mode <host>` before enabling a blocking gate.
