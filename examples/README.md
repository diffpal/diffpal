# DiffPal Examples

Use these examples as copy-paste starting points. Pick one provider auth setup
and one CI system.

The examples use npm `@latest` for quick onboarding. For production, pin
`@diffpal/diffpal`, `diffpal-version`, `@openai/codex`,
`@normahq/codex-acp-bridge`, and `@github/copilot` to versions you have tested.

## Provider Configs

| Setup | Config | CI secret |
| --- | --- | --- |
| Codex API key | [`configs/codex-api-key/config.yaml`](configs/codex-api-key/config.yaml) | `OPENAI_API_KEY` |
| Codex subscription auth | [`configs/codex-subscription/config.yaml`](configs/codex-subscription/config.yaml) | `CODEX_AUTH_JSON_B64` |
| Copilot fine-grained PAT | [`configs/copilot-github-token/config.yaml`](configs/copilot-github-token/config.yaml) | `COPILOT_GITHUB_TOKEN` |

Copy the selected config to `.config/diffpal/config.yaml`.

## CI Examples

| CI system | Codex API key | Codex subscription | Copilot token |
| --- | --- | --- | --- |
| GitHub Actions | [`ci/github-actions/codex-api-key.yml`](ci/github-actions/codex-api-key.yml) | [`ci/github-actions/codex-subscription.yml`](ci/github-actions/codex-subscription.yml) | [`ci/github-actions/copilot-github-token.yml`](ci/github-actions/copilot-github-token.yml) |
| GitLab CI | [`ci/gitlab/codex-api-key.yml`](ci/gitlab/codex-api-key.yml) | [`ci/gitlab/codex-subscription.yml`](ci/gitlab/codex-subscription.yml) | [`ci/gitlab/copilot-github-token.yml`](ci/gitlab/copilot-github-token.yml) |
| Azure Pipelines | [`ci/azure-pipelines/codex-api-key.yml`](ci/azure-pipelines/codex-api-key.yml) | [`ci/azure-pipelines/codex-subscription.yml`](ci/azure-pipelines/codex-subscription.yml) | [`ci/azure-pipelines/copilot-github-token.yml`](ci/azure-pipelines/copilot-github-token.yml) |

## Auth Notes

- Codex API key uses `codex login --with-api-key` with `OPENAI_API_KEY`.
- Codex subscription auth restores an existing `~/.codex/auth.json` from
  `CODEX_AUTH_JSON_B64`. Use it only in trusted same-repository CI.
- Copilot token auth uses `COPILOT_GITHUB_TOKEN`. It must be a fine-grained
  GitHub PAT v2 with the Copilot Requests permission. Classic PATs are not
  supported by the Copilot CLI.
