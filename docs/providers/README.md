# Providers

Use this section to choose and configure the review provider that DiffPal runs
inside your CI job.

DiffPal does not own or manage third-party provider accounts. Create, license,
authenticate, and secure the provider account or CLI with that provider's own
tools.

## Concepts

| Term | Meaning |
| --- | --- |
| Code host | The system that owns pull requests or merge requests, such as GitHub, GitLab, or Azure DevOps. |
| CI system | The runner that checks out the repository and executes DiffPal, such as GitHub Actions, GitLab CI, Azure Pipelines, or a custom runner. |
| Provider | The configured runtime entry that DiffPal asks to perform review reasoning. |
| Agent | The provider-backed CLI or ACP-compatible process that inspects the diff and returns structured review output. |

The code host decides where feedback is published. The CI system decides how
commands and secrets run. The provider or agent decides which model, account,
tools, sandbox, and credentials are used for review.

## Provider Selection

DiffPal selects one provider by matching `diffpal.provider` to an entry under
`runtime.providers`:

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

The selected provider ID must exist in `runtime.providers`. Profiles and the
`DIFFPAL_PROVIDER` environment variable can override the selected provider for a
specific CI job.

## Choose A Provider

| Provider path | Use when | Setup name | Config example |
| --- | --- | --- | --- |
| [Codex](codex.md) | You want the default copy-paste onboarding path or an existing Codex auth file in trusted CI. | `codex-api-key` or `codex-subscription` | [`examples/configs/codex-api-key/config.yaml`](../../examples/configs/codex-api-key/config.yaml) |
| [Copilot](copilot.md) | Your organization already uses Copilot and can provide a supported Copilot token to CI. | `copilot-github-token` | [`examples/configs/copilot-github-token/config.yaml`](../../examples/configs/copilot-github-token/config.yaml) |
| [OpenCode](opencode.md) | You want DiffPal to run through an OpenCode ACP provider already installed and authenticated in CI. | `opencode-acp` | [`examples/configs/opencode-acp/config.yaml`](../../examples/configs/opencode-acp/config.yaml) |
| [Custom ACP CLI](custom-acp.md) | You have another CLI that can start an ACP stdio server. | `generic-acp` | [`examples/configs/generic-acp/config.yaml`](../../examples/configs/generic-acp/config.yaml) |

These setup names are accepted by:

```bash
diffpal init --wizard --setup <setup-name> --platform github
```

Use the provider page for install and authentication, then use the
[Integrations](../integrations/README.md) section for host-specific CI syntax.
