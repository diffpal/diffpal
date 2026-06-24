# OpenCode Provider

## When To Use This Provider

Use OpenCode when your CI environment already installs and authenticates an
OpenCode CLI that can run as the selected DiffPal provider.

## Prerequisites

- A DiffPal config committed at `.config/diffpal/config.yaml`.
- An OpenCode CLI available in the CI job.
- Provider credentials configured with OpenCode's own authentication flow.

## Installation

Install OpenCode in CI using the installation method you use for OpenCode in
that environment. Pin the OpenCode package or source revision in your CI setup
the same way you pin other provider CLIs.

## Authentication In CI

Authenticate OpenCode before running DiffPal. Store any OpenCode credentials in
protected CI secrets and pass them to the OpenCode CLI using its supported
authentication mechanism.

## Minimal Verified Configuration

Use [`examples/configs/opencode-acp/config.yaml`](../../examples/configs/opencode-acp/config.yaml).

The provider selection is:

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

DiffPal does not manage OpenCode accounts, credentials, models, or sandbox
settings. Keep OpenCode credentials in protected CI secrets and run
secret-backed review only in trusted branches, same-repository pull requests, or
maintainer-approved jobs that do not execute untrusted code with secrets.

## Common Failures

- The OpenCode CLI is not installed before DiffPal runs.
- OpenCode authentication is missing from the trusted CI job.
- The configured model is not available to the authenticated OpenCode account.
- The selected `diffpal.provider` does not match the `opencode-acp` provider
  entry.

## Links To Complete CI Examples

- [OpenCode ACP config](../../examples/configs/opencode-acp/config.yaml)
- [Integration guides](../integrations/README.md)
