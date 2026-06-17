# DiffPal Examples

Use these examples as copy-paste starting points. Pick a provider recipe and one
CI system, or adapt the generic ACP template for your own CLI.

The examples pin npm package versions so credentialed CI jobs do not execute
newly published package versions automatically. Update `@diffpal/diffpal`,
`diffpal-version`, `@openai/codex`, `@normahq/codex-acp-bridge`, and
`@github/copilot` intentionally after testing. If you use OpenCode, pin the
OpenCode package or install source in your own CI setup the same way.

## Provider Recipes

| Setup | Config | CI secret |
| --- | --- | --- |
| Generic ACP CLI | [`configs/generic-acp/config.yaml`](configs/generic-acp/config.yaml) | provider-specific |
| Codex API key | [`configs/codex-api-key/config.yaml`](configs/codex-api-key/config.yaml) | `OPENAI_API_KEY` |
| Codex subscription auth | [`configs/codex-subscription/config.yaml`](configs/codex-subscription/config.yaml) | `CODEX_AUTH_JSON_B64` |
| Copilot fine-grained PAT | [`configs/copilot-github-token/config.yaml`](configs/copilot-github-token/config.yaml) | `COPILOT_GITHUB_TOKEN` |
| OpenCode ACP | [`configs/opencode-acp/config.yaml`](configs/opencode-acp/config.yaml) | OpenCode-specific |

Copy the selected config to `.config/diffpal/config.yaml`. To use another ACP
CLI, start from the generic ACP config and replace `generic_acp.cmd` with the
command that starts your provider's ACP stdio server.

## CI Examples

| CI system | Codex API key | Codex subscription | Copilot token |
| --- | --- | --- | --- |
| GitHub Actions | [`ci/github-actions/codex-api-key.yml`](ci/github-actions/codex-api-key.yml) | [`ci/github-actions/codex-subscription.yml`](ci/github-actions/codex-subscription.yml) | [`ci/github-actions/copilot-github-token.yml`](ci/github-actions/copilot-github-token.yml) |
| GitLab CI | [`ci/gitlab/codex-api-key.yml`](ci/gitlab/codex-api-key.yml) | [`ci/gitlab/codex-subscription.yml`](ci/gitlab/codex-subscription.yml) | [`ci/gitlab/copilot-github-token.yml`](ci/gitlab/copilot-github-token.yml) |
| Azure Pipelines | [`ci/azure-pipelines/codex-api-key.yml`](ci/azure-pipelines/codex-api-key.yml) | [`ci/azure-pipelines/codex-subscription.yml`](ci/azure-pipelines/codex-subscription.yml) | [`ci/azure-pipelines/copilot-github-token.yml`](ci/azure-pipelines/copilot-github-token.yml) |

## Auth Notes

- Generic ACP CLI auth is provider-specific. Install and authenticate the CLI in
  CI before running DiffPal, then point `generic_acp.cmd` at its ACP command.
- Codex API key uses `codex login --with-api-key` with `OPENAI_API_KEY`.
- Codex subscription auth restores an existing `~/.codex/auth.json` from
  `CODEX_AUTH_JSON_B64`. Generate the secret with the recipe below and use it
  only in trusted same-repository CI.
- Copilot token auth uses `COPILOT_GITHUB_TOKEN`. It must be a fine-grained
  GitHub PAT v2 with the Copilot Requests permission. Classic PATs are not
  supported by the Copilot CLI.
- OpenCode ACP uses the `opencode acp` command resolved by the runtime. Install
  and authenticate OpenCode before running DiffPal.
- The GitLab examples restrict secret-backed jobs to same-project merge
  requests, require a protected `DIFFPAL_TRUSTED_REVIEW=true` variable, and run
  as manual jobs so maintainers decide when provider credentials are exposed.
  The Azure examples skip credentialed review steps when
  `System.PullRequest.IsFork` is `True`.

To use another ACP CLI, copy the closest CI example for your host and replace
only the provider install/auth step plus `.config/diffpal/config.yaml`.

## Generate CODEX_AUTH_JSON_B64

Use this only when you intentionally want Codex subscription auth in trusted CI.
API-key auth is simpler to rotate and should remain the default automation path.

This command creates a fresh file-backed Codex login in a temporary `CODEX_HOME`,
then base64-encodes the generated `auth.json` for the `CODEX_AUTH_JSON_B64`
secret:

```bash
npm install --global @openai/codex@0.139.0

tmp_codex_home="$(mktemp -d)"
trap 'rm -rf "$tmp_codex_home"' EXIT

printf '%s\n' 'cli_auth_credentials_store = "file"' > "$tmp_codex_home/config.toml"
CODEX_HOME="$tmp_codex_home" codex login --device-auth

test -s "$tmp_codex_home/auth.json"
codex_auth_json_b64="$(base64 < "$tmp_codex_home/auth.json" | tr -d '\n')"

gh secret set CODEX_AUTH_JSON_B64 --body "$codex_auth_json_b64"
```

For GitLab or Azure Pipelines, store the value of `codex_auth_json_b64` as a
masked/protected `CODEX_AUTH_JSON_B64` CI secret or pipeline variable.

Treat `auth.json` and `CODEX_AUTH_JSON_B64` like passwords. They contain access
tokens. Do not commit them, paste them into issues, or expose them to fork PR/MR
jobs.
