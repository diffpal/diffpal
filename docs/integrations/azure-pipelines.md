# Azure Pipelines

Use this page to run DiffPal in Azure Pipelines pull request validation.

## Supported Outputs

- Azure PR summary thread.
- File-bound PR threads for actionable findings.
- Azure PR status named `DiffPal Review`.

## Prerequisites

- An Azure Repos project with PR validation.
- The DiffPal Review extension installed, or `diffpal` available in the job.
- A committed DiffPal config at `.config/diffpal/config.yaml`.
- A provider secret such as `OPENAI_API_KEY`.
- Pipeline access to `System.AccessToken`.

See [Shared Setup](README.md#shared-setup) and
[Provider Recipes](README.md#provider-recipes).

## Required Checkout Behavior

Use full checkout history:

```yaml
- checkout: self
  fetchDepth: 0
```

When `base` and `head` are omitted, `DiffPalReview@1` uses PR metadata and the
target branch to compute the merge base.

## Required Token And Minimum Permissions

Enable **Allow scripts to access the OAuth token** and pass
`SYSTEM_ACCESSTOKEN` to the review task:

```yaml
env:
  SYSTEM_ACCESSTOKEN: $(System.AccessToken)
```

Use `SYSTEM_ACCESSTOKEN` for pipeline-scoped access. Use
`AZURE_DEVOPS_EXT_PAT` only when your organization requires a dedicated PAT.

## Provider Installation And Authentication

For the Codex API-key recipe:

```yaml
- task: UseNode@1
  inputs:
    version: "22.x"

- script: npm install --global @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3
  displayName: Install Codex provider

- script: printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
  displayName: Authenticate Codex
  env:
    OPENAI_API_KEY: $(OPENAI_API_KEY)
```

For other providers, replace only the install/auth steps and matching config.
See [Provider Recipes](README.md#provider-recipes).

## Minimal Pipeline

```yaml
trigger: none
pr:
  - main

pool:
  vmImage: ubuntu-latest

steps:
  - checkout: self
    fetchDepth: 0

  - task: UseNode@1
    inputs:
      version: "22.x"

  - script: npm install --global @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3
    displayName: Install Codex provider

  - script: printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
    displayName: Authenticate Codex
    condition: and(succeeded(), ne(variables['System.PullRequest.IsFork'], 'True'))
    env:
      OPENAI_API_KEY: $(OPENAI_API_KEY)

  - task: DiffPalReview@1
    displayName: DiffPal review
    condition: and(succeeded(), ne(variables['System.PullRequest.IsFork'], 'True'))
    inputs:
      diffpalVersion: latest
      profile: ci
      feedback: review
      gate: true
    env:
      OPENAI_API_KEY: $(OPENAI_API_KEY)
      SYSTEM_ACCESSTOKEN: $(System.AccessToken)
```

## Feedback Modes

Use `feedback: review` for status, summary thread, and PR threads. Use
`feedback: summary` for status and summary thread without file-bound PR
threads.

See [Feedback Modes](README.md#feedback-modes).

## Merge-Gate Setup

Set `gate: true` on `DiffPalReview@1`. Blocking findings fail the task and set
the Azure PR status to failed.

See [Merge Gates](README.md#merge-gates).

## Fork Or Untrusted-Contribution Behavior

Keep credentialed review steps behind:

```yaml
condition: and(succeeded(), ne(variables['System.PullRequest.IsFork'], 'True'))
```

Use stricter organization-specific trusted-source conditions when needed. See
[Untrusted Contributions](README.md#untrusted-contributions).

## Expected Results

- Azure PR threads for actionable findings when feedback is `review`.
- A PR summary thread headed `DiffPal Review Summary`.
- An Azure PR status named `DiffPal Review`.
- Failed task only for blocking gated findings or incomplete review setup.

## Common Failures

- `fetchDepth: 0` is missing.
- **Allow scripts to access the OAuth token** is disabled.
- `SYSTEM_ACCESSTOKEN` is not passed to the review task.
- Provider variables are unavailable in fork PR validation.

See [Common Failures](README.md#common-failures).

## Related Examples

- [Codex API key](../../examples/ci/azure-pipelines/codex-api-key.yml)
- [Codex subscription auth](../../examples/ci/azure-pipelines/codex-subscription.yml)
- [Copilot token](../../examples/ci/azure-pipelines/copilot-github-token.yml)
