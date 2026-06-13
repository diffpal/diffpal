# DiffPal

DiffPal reviews pull request diffs and publishes policy-aware feedback back to
your CI system.

It is built for teams that want AI review output that is easy to scan:

- PR summaries that explain what changed
- inline comments only for actionable findings
- merge gates through checks/statuses, not bot approvals
- one config file that works across GitHub, GitLab, and Azure DevOps

## Quick Start

Add a DiffPal config, add a provider secret, then choose the CI example for your
platform.

The examples use npm `@latest` for quick onboarding. For production, pin
`@diffpal/diffpal`, `diffpal-version`, `@openai/codex`, and
`@normahq/codex-acp-bridge` to versions you have tested.

## Config

Commit `.config/diffpal/config.yaml`:

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
      # - best-practices
```

Add `OPENAI_API_KEY` as a CI secret so the Codex CLI can act as the
review provider. Platform publish tokens are CI-specific:

| Platform | Publish token |
| --- | --- |
| GitHub Actions | built-in `GITHUB_TOKEN` |
| GitLab CI | built-in `CI_JOB_TOKEN` or `GITLAB_TOKEN` |
| Azure Pipelines | built-in `SYSTEM_ACCESSTOKEN` |

## GitHub Actions

Create `.github/workflows/diffpal-review.yml`.

The action installs the DiffPal CLI. The workflow installs only the Codex
provider command.

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
        run: npm install --global @openai/codex@latest @normahq/codex-acp-bridge@latest

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
          feedback: balanced
          gate: true
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

The same-repository PR guard keeps provider secrets out of untrusted fork
workflows. Remove or change that guard only after designing a fork-safe release
flow.

## GitLab CI

Add this job to `.gitlab-ci.yml`.

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
    - npm install --global @diffpal/diffpal@latest @openai/codex@latest @normahq/codex-acp-bridge@latest
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

Set `OPENAI_API_KEY` as a protected/masked CI variable. Use the built-in
`CI_JOB_TOKEN` when your GitLab instance allows it, or set `GITLAB_TOKEN` for a
dedicated API token.

## Azure Pipelines

Enable **Allow scripts to access the OAuth token**, then add this to
`azure-pipelines.yml`.

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

  - script: npm install --global @openai/codex@latest @normahq/codex-acp-bridge@latest
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
      feedback: balanced
      gate: true
    env:
      OPENAI_API_KEY: $(OPENAI_API_KEY)
      SYSTEM_ACCESSTOKEN: $(System.AccessToken)
```

The Azure task installs the DiffPal CLI by default. Set `install: false` to use
a preinstalled binary from `PATH`, or set `diffpalPath` to a custom binary path.

## What You Should See

On pull requests, DiffPal can publish:

- a review summary with a semantic overview of the change
- a check/status for merge gating
- inline comments or threads for actionable findings
- JSON, SARIF, and CI artifacts for later inspection

The default review checks are `security`, `bugs`, `performance`, and
`best-practices`. The default review language is English. Checks, language, and
custom review instructions are configurable in `.config/diffpal/config.yaml` or
by CLI flags such as `--review-checks`, `--instructions`, and
`--instructions-file`.

## Local Debugging

Local commands are useful for setup checks and debugging, but they are not the
main CI setup path.

```bash
npm install --global @diffpal/diffpal@latest @openai/codex@latest @normahq/codex-acp-bridge@latest
printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
diffpal init
diffpal doctor --mode github
diffpal review local --base origin/main --head HEAD
```

## Documentation

- [Quickstart](docs/quickstart.md)
- [CI setup guide](docs/ci-examples.md)
- [Config reference](docs/config-reference.md)
- [Findings schema](docs/findings-schema.md)
- [GitLab adapter reference](docs/platform-gitlab.md)
- [Azure adapter reference](docs/platform-azure.md)
- [Release process](docs/release.md)
- [Contributing](CONTRIBUTING.md)
