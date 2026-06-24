# Copilot Provider

## When To Use This Provider

Use Copilot when your organization already uses Copilot and can provide a
supported Copilot token to the CI job that runs DiffPal.

## Prerequisites

- A DiffPal config committed at `.config/diffpal/config.yaml`.
- Node.js available in the CI job.
- A `COPILOT_GITHUB_TOKEN` CI secret for the Copilot CLI.

## Installation

Install the Copilot CLI in CI:

```bash
npm install --global @github/copilot@1.0.61
```

## Authentication In CI

Store `COPILOT_GITHUB_TOKEN` as a protected CI secret. The Copilot CLI reads
`COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, then `GITHUB_TOKEN`; use
`COPILOT_GITHUB_TOKEN` for DiffPal so the provider token stays separate from
the platform publishing token.

## Minimal Verified Configuration

Use [`examples/configs/copilot-github-token/config.yaml`](../../examples/configs/copilot-github-token/config.yaml).

The provider selection is:

```yaml
runtime:
  providers:
    copilot-acp:
      type: copilot_acp
      copilot_acp:
        model: auto

diffpal:
  provider: copilot-acp
```

## How To Test Provider Connectivity

Validate the local runtime first:

```bash
diffpal doctor --profile ci --mode local
```

Then run a provider-backed smoke review on a trusted branch:

```bash
diffpal --profile ci review local \
  --base origin/main \
  --head HEAD \
  --feedback summary \
  --out .artifacts/diffpal/findings.json
```

## Security Considerations

Do not reuse the platform publishing token as the Copilot provider token. Keep
`COPILOT_GITHUB_TOKEN` in protected CI secrets and run secret-backed review only
in trusted branches, same-repository pull requests, or maintainer-approved jobs
that do not execute untrusted code with secrets.

## Common Failures

- The Copilot CLI is not installed before DiffPal runs.
- `COPILOT_GITHUB_TOKEN` is missing from the trusted CI job.
- The token lacks the Copilot Requests permission required by the Copilot CLI.
- A classic PAT is used where the Copilot CLI requires a fine-grained token.
- The selected `diffpal.provider` does not match the `copilot-acp` provider
  entry.

## Links To Complete CI Examples

- [GitHub Actions with Copilot token](../../examples/ci/github-actions/copilot-github-token.yml)
- [GitLab CI with Copilot token](../../examples/ci/gitlab/copilot-github-token.yml)
- [Azure Pipelines with Copilot token](../../examples/ci/azure-pipelines/copilot-github-token.yml)
