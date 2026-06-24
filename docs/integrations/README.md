# Integrations

Use this section to run DiffPal in CI and publish review feedback to your code
host. Host-specific pages all follow the same shape:

- [GitHub Actions](github-actions.md)
- [GitLab CI](gitlab-ci.md)
- [Azure Pipelines](azure-pipelines.md)
- [Custom CI/CD](custom-ci.md)

Copy-paste configs and pipelines live in [`examples/`](../../examples/README.md).
Use the [GitHub quickstart](../getting-started/github-quickstart.md) when you
want the shortest first setup path.

## Shared Setup

Every host needs:

1. Full git history for the reviewed pull request or merge request.
2. A DiffPal config committed at `.config/diffpal/config.yaml`.
3. The provider CLI runtime required by the selected agent.
4. A provider auth secret.
5. A platform token with permission to publish review feedback.

For Jenkins, Buildkite, CircleCI, Bitbucket Pipelines, internal runners, or any
other CI system, use the [Custom CI/CD guide](custom-ci.md).

## Provider Recipes

Choose a ready-made provider recipe or configure `generic_acp` for your own ACP
CLI. The selected provider lives under `runtime.providers`; the host page only
changes the CI syntax for install, auth, checkout, and publishing.

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

The rest of the workflow stays the same: full checkout, DiffPal config,
provider secret, platform token, feedback mode, and optional gate.

## Feedback Modes

Use `feedback` for normal setup:

| Feedback | Behavior |
| --- | --- |
| `summary` | PR/MR summary plus non-file artifacts such as status, SARIF, or Code Quality. No file-level findings are published. |
| `review` | Summary plus file-level comments, threads, or discussions for the platform. Non-blocking findings remain visible without becoming merge blockers. |

Default review publish surfaces:

| Platform | Default surfaces |
| --- | --- |
| GitHub | `comments,sarif,summary` |
| GitLab | `code-quality,discussions,status,sarif,summary` |
| Azure | `threads,status,summary` |

Common artifacts are listed in the [artifacts reference](../reference/artifacts.md).

## Merge Gates

Enable `gate` when blocking findings should fail the CI job. Start with
`block_on: high`; lower the threshold only after tuning review policy. See the
[configuration gate reference](../reference/configuration.md#gate).

Tooling failures such as setup, provider auth, diff collection, or publishing
fail the job because the review result is incomplete, even when the merge gate
is disabled.

## Untrusted Contributions

Keep provider credentials out of untrusted fork pipelines. Run secret-backed
DiffPal review only for trusted branches, same-repository pull requests, or
maintainer-approved workflows that do not execute untrusted code with secrets.

See [Fork Pull Requests And Secrets](../help/troubleshooting.md#fork-pull-requests-and-secrets).

## Common Failures

Most integration failures come from:

- shallow checkout;
- missing provider secret;
- provider CLI not installed or authenticated;
- platform token missing write permission;
- running secret-backed review on an untrusted fork PR.

Use the [troubleshooting guide](../help/troubleshooting.md) for fixes.
