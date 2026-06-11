# CI Examples

## GitHub Actions

```yaml
name: diffpal-review

on:
  pull_request:
    types: [opened, synchronize, reopened, ready_for_review]

concurrency:
  group: diffpal-${{ github.event.pull_request.number }}
  cancel-in-progress: true

jobs:
  review:
    if: ${{ github.event.pull_request.head.repo.full_name == github.repository }}
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
          node-version: 20
      - run: npm install --global @diffpal/diffpal@1.2.3
      - run: >-
          diffpal review github
          --base ${{ github.event.pull_request.base.sha }}
          --head ${{ github.event.pull_request.head.sha }}
          --gate
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

## GitLab CI

```yaml
stages:
  - review

diffpal-review:
  stage: review
  image: node:20
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event"'
  resource_group: "diffpal:$CI_MERGE_REQUEST_IID"
  script:
    - npm install --global @diffpal/diffpal@1.2.3
    - diffpal review gitlab \
        --base "$CI_MERGE_REQUEST_DIFF_BASE_SHA" \
        --head "$CI_COMMIT_SHA" \
        --gate
  artifacts:
    reports:
      codequality: .artifacts/diffpal/codequality.json
      sarif: .artifacts/diffpal/diffpal.sarif
```

## Azure Pipelines

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
      versionSpec: "20.x"
  - script: npm install --global @diffpal/diffpal@1.2.3
    displayName: Install DiffPal CLI
  - task: DiffPalReview@1
    inputs:
      gate: true
    env:
      SYSTEM_ACCESSTOKEN: $(System.AccessToken)
```

## Semantics

- GitHub pipeline publishes check runs, a PR-level summary comment, and inline reviews for actionable findings.
- GitLab pipeline writes both `discussions` and artifact reports.
- Azure pipeline posts PR threads and PR status with merge-policy-compatible names.
- The GitHub and Azure task wrappers expect a `diffpal` binary on `PATH`, typically installed from a pinned `@diffpal/diffpal` SemVer.
- Azure publish requires `Allow scripts to access the OAuth token` so `SYSTEM_ACCESSTOKEN` is populated.
