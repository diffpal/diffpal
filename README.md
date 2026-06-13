# DiffPal

DiffPal reviews pull request diffs and publishes policy-aware feedback back to
your CI system.

It is built for teams that want AI review output that is easy to scan:

- PR summaries that explain what changed
- inline comments only for actionable findings
- merge gates through checks/statuses, not bot approvals
- one config file that works across GitHub, GitLab, and Azure DevOps

## Quick Start

1. Choose a provider config from [`examples/configs`](examples/configs).
2. Copy it to `.config/diffpal/config.yaml`.
3. Add the required CI secret.
4. Copy the matching CI example from [`examples/ci`](examples/ci).

The fastest GitHub setup is:

- config: [`examples/configs/codex-api-key/config.yaml`](examples/configs/codex-api-key/config.yaml)
- workflow: [`examples/ci/github-actions/codex-api-key.yml`](examples/ci/github-actions/codex-api-key.yml)
- secret: `OPENAI_API_KEY`

The examples use npm `@latest` for onboarding. For production, pin
`@diffpal/diffpal`, `diffpal-version`, provider CLIs, and bridge packages to
versions you have tested.

## Supported Setups

| Provider auth | Secret | Notes |
| --- | --- | --- |
| Codex API key | `OPENAI_API_KEY` | Best first setup for CI. |
| Codex subscription auth | `CODEX_AUTH_JSON_B64` | Trusted same-repository CI only. |
| Copilot fine-grained PAT | `COPILOT_GITHUB_TOKEN` | Requires Copilot Requests permission; classic PATs are not supported. |

| CI system | Example directory |
| --- | --- |
| GitHub Actions | [`examples/ci/github-actions`](examples/ci/github-actions) |
| GitLab CI | [`examples/ci/gitlab`](examples/ci/gitlab) |
| Azure Pipelines | [`examples/ci/azure-pipelines`](examples/ci/azure-pipelines) |

## Minimal Config Shape

DiffPal uses the current profile-based config shape:

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
    checks:
      - security
      - bugs
      - performance
      - best-practices

profiles:
  ci:
    diffpal:
      gate:
        block_on: high
```

Use `diffpal.review.instructions` or `--instructions-file` for local prompt
tuning.

## What You Should See

On pull requests, DiffPal can publish:

- a review summary with a semantic overview of the change
- a check/status for merge gating
- inline comments or threads for actionable findings
- JSON, SARIF, and CI artifacts for later inspection

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

- [Examples](examples/README.md)
- [Quickstart](docs/quickstart.md)
- [CI setup guide](docs/ci-examples.md)
- [Config reference](docs/config-reference.md)
- [Findings schema](docs/findings-schema.md)
- [GitLab adapter reference](docs/platform-gitlab.md)
- [Azure adapter reference](docs/platform-azure.md)
- [Release process](docs/release.md)
- [Contributing](CONTRIBUTING.md)
