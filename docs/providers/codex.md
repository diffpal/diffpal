# Codex Provider

## When To Use This Provider

Use Codex when you want the default DiffPal onboarding path, a copy-paste Codex
API-key setup, or trusted CI that restores an existing Codex auth file.

## Prerequisites

- A DiffPal config committed at `.config/diffpal/config.yaml`.
- Node.js available in the CI job.
- Either an `OPENAI_API_KEY` CI secret or a trusted `CODEX_AUTH_JSON_B64` CI
  secret generated from your own Codex login.

## Installation

Install the Codex CLI and ACP bridge in CI:

```bash
npm install --global @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3
```

## Authentication In CI

For API-key auth, store `OPENAI_API_KEY` as a protected CI secret and run:

Do not expose `OPENAI_API_KEY` to untrusted fork jobs. Keep the credentialed
review job limited to trusted branches, same-repository pull requests, or
maintainer-approved jobs that do not execute fork-controlled code.

```bash
printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
```

For subscription auth, generate `CODEX_AUTH_JSON_B64` with the recipe in
[`examples/README.md`](../../examples/README.md#generate-codexauthjsonb64),
then restore it only in trusted CI:

Do not expose restored Codex auth files to untrusted fork jobs.

```bash
mkdir -p "$HOME/.codex"
printf '%s' "$CODEX_AUTH_JSON_B64" | base64 --decode > "$HOME/.codex/auth.json"
chmod 600 "$HOME/.codex/auth.json"
```

## Minimal Verified Configuration

Use [`examples/configs/codex-api-key/config.yaml`](../../examples/configs/codex-api-key/config.yaml)
for API-key auth or
[`examples/configs/codex-subscription/config.yaml`](../../examples/configs/codex-subscription/config.yaml)
for subscription auth.

The provider selection is:

```yaml
runtime:
  providers:
    codex-acp:
      type: codex_acp
      codex_acp:
        reasoning_effort: low

diffpal:
  provider: codex-acp
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

Do not expose `OPENAI_API_KEY` or restored Codex auth files to untrusted fork
jobs. Keep Codex credentials in protected CI secrets and run secret-backed
review only in trusted branches, same-repository pull requests, or
maintainer-approved jobs that do not execute untrusted code with secrets.

## Common Failures

- The Codex CLI or ACP bridge is not installed before DiffPal runs.
- `OPENAI_API_KEY` or `CODEX_AUTH_JSON_B64` is missing from the trusted CI job.
- Subscription auth was generated from an expired or invalid local login.
- The selected `diffpal.provider` does not match the `codex-acp` provider entry.

## Links To Complete CI Examples

- [GitHub Actions with Codex API key](../../examples/ci/github-actions/codex-api-key.yml)
- [GitHub Actions with Codex subscription auth](../../examples/ci/github-actions/codex-subscription.yml)
- [GitLab CI with Codex API key](../../examples/ci/gitlab/codex-api-key.yml)
- [GitLab CI with Codex subscription auth](../../examples/ci/gitlab/codex-subscription.yml)
- [Azure Pipelines with Codex API key](../../examples/ci/azure-pipelines/codex-api-key.yml)
- [Azure Pipelines with Codex subscription auth](../../examples/ci/azure-pipelines/codex-subscription.yml)

Next step: choose the host-specific CI example that matches your code host.
