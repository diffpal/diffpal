# Custom ACP-Compatible CLI

## When To Use This Provider

Use a custom ACP-compatible CLI when you have another agent process that can
start an ACP stdio server and you want DiffPal to keep the same review,
artifact, publishing, and gate workflow.

## Prerequisites

- A DiffPal config committed at `.config/diffpal/config.yaml`.
- A CLI that can start an ACP stdio server.
- Provider-specific credentials configured in trusted CI.

## Installation

Install your ACP-compatible CLI before running DiffPal. Pin the package,
container image, or source revision in CI so review behavior is repeatable.

## Authentication In CI

Authenticate the CLI with its own supported mechanism before running DiffPal.
Store provider credentials in protected CI secrets and do not commit them to the
DiffPal config.

Do not expose provider credentials to untrusted fork jobs. Keep the
credentialed review job limited to trusted branches, same-repository pull
requests, or maintainer-approved jobs that do not execute fork-controlled code.

## Minimal Verified Configuration

Use [`examples/configs/generic-acp/config.yaml`](../../examples/configs/generic-acp/config.yaml)
as the starting point and replace `generic_acp.cmd` with the command that
starts your provider's ACP stdio server:

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

## Expected Result

The smoke review should complete, write
`.artifacts/diffpal/findings.json`, and print a Markdown summary to stdout.

## Security Considerations

DiffPal delegates provider-specific model, account, tool, sandbox, and approval
policy to the ACP CLI you configure. Keep credentials in protected CI secrets
and run secret-backed review only in trusted branches, same-repository pull
requests, or maintainer-approved jobs that do not execute untrusted code with
secrets.

## Common Failures

- `generic_acp.cmd` does not start an ACP stdio server.
- The CLI is not installed in the CI job.
- Required provider-specific credentials are missing.
- The configured command requires interactive input in CI.
- The selected `diffpal.provider` does not match the provider ID under
  `runtime.providers`.

## Links To Complete CI Examples

- [Generic ACP config](../../examples/configs/generic-acp/config.yaml)
- [Custom CI/CD guide](../integrations/custom-ci.md)

Next step: replace `generic_acp.cmd` with your ACP command, then run the
portable [Custom CI/CD](../integrations/custom-ci.md) flow.
