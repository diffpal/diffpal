# Integrations

DiffPal's portability point is provider install and auth: CI chooses and
authenticates the provider, while DiffPal keeps the PR review workflow,
artifacts, publishing behavior, and gates consistent across hosts.

Copy-paste files live in [`examples/`](../../examples/README.md). Use the
[GitHub quickstart](../getting-started/github-quickstart.md) for the fastest
GitHub path.

## Common Setup

Every CI system needs:

1. A full git checkout, so DiffPal can compare base and head commits.
2. The provider CLI runtime required by your selected agent.
3. A DiffPal config committed at `.config/diffpal/config.yaml`.
4. A provider auth secret.
5. A platform token so DiffPal can publish PR feedback.

Choose a ready-made provider recipe or configure `generic_acp` for your own ACP
CLI. The selected provider lives under `runtime.providers`; the CI-specific
steps are the install and authentication commands for that provider.

| Setup | Config | Required secret |
| --- | --- | --- |
| Generic ACP CLI | [`examples/configs/generic-acp/config.yaml`](../../examples/configs/generic-acp/config.yaml) | provider-specific |
| Codex API key | [`examples/configs/codex-api-key/config.yaml`](../../examples/configs/codex-api-key/config.yaml) | `OPENAI_API_KEY` |
| Codex subscription auth | [`examples/configs/codex-subscription/config.yaml`](../../examples/configs/codex-subscription/config.yaml) | `CODEX_AUTH_JSON_B64` |
| Copilot token | [`examples/configs/copilot-github-token/config.yaml`](../../examples/configs/copilot-github-token/config.yaml) | `COPILOT_GITHUB_TOKEN` |
| OpenCode ACP | [`examples/configs/opencode-acp/config.yaml`](../../examples/configs/opencode-acp/config.yaml) | OpenCode-specific |

These setup names are accepted by `diffpal init --wizard --setup ...`.

## Using Another ACP CLI

DiffPal can use any CLI that starts an ACP stdio server. Copy the CI example
for your host, replace the provider install/authentication step with your CLI's
setup, and use a config like:

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

The rest of the GitHub, GitLab, and Azure examples stay the same: checkout the
full git history, run DiffPal with the selected profile, and pass the platform
token for publishing feedback.

OpenCode is available as a first-class ACP alias:

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

Install and authenticate `opencode` in CI before the DiffPal step.

## Host Guides

- [GitHub Actions](github-actions.md)
- [GitLab CI](gitlab-ci.md)
- [Azure Pipelines](azure-pipelines.md)

## Feedback And Outputs

Use `feedback` for normal setup:

| Feedback | Behavior |
| --- | --- |
| `summary` | PR/MR summary plus non-file artifacts such as status, SARIF, or Code Quality. No file-level findings are published. |
| `review` | Summary plus file-level comments, threads, or discussions for the platform. Non-blocking findings remain visible without becoming merge blockers. |

The semantic change overview is shown by default in PR reviews. Turn it off
with `summary-overview: false` in GitHub Actions or `--summary-overview=false`
on the CLI.

Default review publish surfaces:

| Platform | Default surfaces |
| --- | --- |
| GitHub | `comments,sarif,summary` |
| GitLab | `code-quality,discussions,status,sarif,summary` |
| Azure | `threads,status,summary` |

Common artifacts are listed in the [artifacts reference](../reference/artifacts.md).

## Production Hardening

- Pin npm package versions after the first successful setup.
- Keep provider secrets out of untrusted fork pipelines.
- Start with `block_on: high`; lower the threshold only after tuning policy.
- Keep `fetch-depth: 0`, `GIT_DEPTH: "0"`, or `fetchDepth: 0` in CI.
- Run `diffpal doctor --mode <host>` before enabling a blocking gate.

For common failures and host-specific fixes, see
[troubleshooting](../help/troubleshooting.md).
