# CI Setup Guide

This is the main setup guide for running DiffPal in pull request pipelines.
The default provider path is Codex ACP.

The examples use npm `@latest` for quick onboarding. For production, pin
`@diffpal/diffpal`, `diffpal-version`, and `@openai/codex` to versions you
have tested.

## Common Setup

Every CI system needs:

1. A full git checkout, so DiffPal can compare base and head commits.
2. Node.js, because the Codex provider is installed with npm.
3. A DiffPal config committed at `.config/diffpal/config.yaml`.
4. `OPENAI_API_KEY` as a secret for the Codex provider.
5. A platform token so DiffPal can publish PR feedback.

Commit `.config/diffpal/config.yaml` with the provider and review gate:

```yaml
version: v1

runtime:
  providers:
    codex-acp:
      type: codex_acp
      codex_acp:
        reasoning_effort: low

diffpal:
  provider: codex-acp
  gate:
    block_on: high
  review:
    language: en
    instructions: |
      Prefer actionable findings that are directly supported by the diff.
    checks:
      - security
      - bugs
      - performance
      - best-practices
```

You can generate a starting config locally with `diffpal init`, then commit the
reviewed file. `diffpal doctor --mode <host>` is useful as a preflight or
troubleshooting command, but it is not required in the normal PR workflow.

## GitHub Actions

### Required Secrets and Permissions

| Name | Required | Purpose |
| --- | --- | --- |
| `OPENAI_API_KEY` | yes | Authenticates the Codex CLI provider. |
| `GITHUB_TOKEN` | built in | Publishes check runs, summary comments, and inline comments. |

Workflow permissions:

```yaml
permissions:
  contents: read
  pull-requests: write
  checks: write
```

Use a same-repository PR guard before exposing `OPENAI_API_KEY`.

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

      - name: Install Codex provider
        run: npm install --global @openai/codex@latest

      - name: Authenticate Codex
        run: printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}

      - name: Review pull request
        uses: diffpal/diffpal@v0.1.2
        with:
          diffpal-version: latest
          base: ${{ github.event.pull_request.base.sha }}
          head: ${{ github.event.pull_request.head.sha }}
          repo: ${{ github.repository }}
          review-id: github-pr-${{ github.event.pull_request.number }}
          language: en
          review-checks: security,bugs,performance,best-practices
          instructions-file: .config/diffpal/review-instructions.md
          feedback: balanced
          summary-overview: true
          gate: true
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
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
- To change the CLI version, set `diffpal-version` on the action step.
- To hide the change overview section, set `summary-overview: false`.
- Fork PRs do not run: this is intentional when using secrets.

## GitLab CI

### Required Variables

| Name | Required | Purpose |
| --- | --- | --- |
| `OPENAI_API_KEY` | yes | Authenticates the Codex CLI provider. |
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
    - npm install --global @diffpal/diffpal@latest @openai/codex@latest
    - printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
  script:
    - >-
      diffpal review gitlab
      --base "$CI_MERGE_REQUEST_DIFF_BASE_SHA"
      --head "$CI_COMMIT_SHA"
      --repo "$CI_PROJECT_PATH"
      --review-id "gitlab-mr-$CI_MERGE_REQUEST_IID"
      --language en
      --review-checks security,bugs,performance,best-practices
      --instructions-file .config/diffpal/review-instructions.md
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
| `OPENAI_API_KEY` | yes | Authenticates the Codex CLI provider. |
| `SYSTEM_ACCESSTOKEN` | yes | Publishes Azure PR threads and status. |

In Azure Pipelines, enable **Allow scripts to access the OAuth token** so
`$(System.AccessToken)` is available to the task.

The `DiffPalReview@1` task installs the DiffPal CLI by default.

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

  - script: npm install --global @openai/codex@latest
    displayName: Install Codex provider

  - script: printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
    displayName: Authenticate Codex
    env:
      OPENAI_API_KEY: $(OPENAI_API_KEY)

  - task: DiffPalReview@1
    displayName: DiffPal review
    inputs:
      diffpalVersion: latest
      language: en
      reviewChecks: security,bugs,performance,best-practices
      instructionsFile: .config/diffpal/review-instructions.md
      feedback: balanced
      gate: true
    env:
      OPENAI_API_KEY: $(OPENAI_API_KEY)
      SYSTEM_ACCESSTOKEN: $(System.AccessToken)
```

### What You Should See

- Azure PR threads for actionable findings.
- An Azure PR summary thread headed `DiffPal Review Summary`.
- Azure PR status named `DiffPal Review`.
- Failed task when `gate` is true and blocking findings exist.

### Common Azure Fixes

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
- Keep `OPENAI_API_KEY` out of untrusted fork pipelines.
- Start with `--block-on high`; lower the threshold only after tuning policy.
- Keep `fetch-depth: 0`, `GIT_DEPTH: "0"`, or `fetchDepth: 0` in CI.
- Run `diffpal doctor --mode <host>` before enabling a blocking gate.
