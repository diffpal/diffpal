# DiffPal Config Reference

DiffPal reads one config file from the repository:

`.config/diffpal/config.yaml`

Generate it with:

```bash
diffpal init
```

## Default Copilot Config

The public onboarding path uses Copilot ACP:

```yaml
version: v1

defaults:
  provider: copilot-acp
  policy: default

providers:
  copilot-acp:
    type: copilot_acp
    copilot_acp: {}

policies:
  default:
    block_on: high

review:
  context_lines: 20
  max_files: 200
  language: en
  checks:
    - bugs
    - performance
    - best-practices
  chunking:
    max_patch_chars: 12000
    max_files_per_chunk: 20

platforms:
  github:
    summary_comment:
      enabled: true
  gitlab: {}
  azure: {}
```

Install the matching provider command in CI:

```bash
npm install --global @github/copilot@latest
```

Set `COPILOT_GITHUB_TOKEN` as a CI secret. Do not commit token values into the
config file.

## Review Settings

| Field | Default | Purpose |
| --- | --- | --- |
| `review.language` | `en` | Language for finding text and summaries. |
| `review.checks` | `bugs`, `performance`, `best-practices` | Review scopes to run. |
| `review.context_lines` | `20` | Neighboring source lines included with each diff hunk. |
| `review.max_files` | `200` | Maximum changed files to review. |
| `review.chunking.max_patch_chars` | `12000` | Maximum context size per model chunk. |
| `review.chunking.max_files_per_chunk` | `20` | Maximum files per model chunk. |

Review checks map to finding categories:

| Check | Categories |
| --- | --- |
| `bugs` | security, correctness, reliability |
| `performance` | performance |
| `best-practices` | maintainability, testing, style |

Override review settings per run:

```bash
diffpal review github \
  --base "$BASE_SHA" \
  --head "$HEAD_SHA" \
  --language en \
  --review-checks bugs,performance,best-practices \
  --feedback balanced
```

## Policy and Gating

`policies.default.block_on` controls which findings are blocking:

```yaml
policies:
  default:
    block_on: high
```

Allowed values:

- `low`
- `medium`
- `high`
- `critical`

Use `--gate` in CI to fail the job when blocking findings exist.

## Platform Auth

DiffPal can read platform tokens from config values, but CI environment
variables are preferred.

| Platform | Preferred env | Config field |
| --- | --- | --- |
| GitHub | `GITHUB_TOKEN` | `platforms.github.auth.token` |
| GitLab | `CI_JOB_TOKEN` or `GITLAB_TOKEN` | `platforms.gitlab.auth.job_token`, `platforms.gitlab.auth.api_token` |
| Azure | `SYSTEM_ACCESSTOKEN` | `platforms.azure.auth.system_access_token` |

Only use envsubst placeholders for values that are guaranteed to exist:

```yaml
platforms:
  github:
    auth:
      token: "${GITHUB_TOKEN}"
```

Missing envsubst variables fail config loading. For optional CI credentials,
omit the config value and let DiffPal read the standard environment variable.

## Alternate Hosted OpenAI Provider

Copilot ACP is the default onboarding provider. If you prefer hosted OpenAI,
switch the provider block:

```yaml
defaults:
  provider: openai-fast

providers:
  openai-fast:
    type: openai
    openai:
      model: "${DIFFPAL_OPENAI_MODEL}"
      api_key: "${OPENAI_API_KEY}"
```

Then set `OPENAI_API_KEY` in CI.

## Environment Overrides

These environment variables override config values:

- `DIFFPAL_PROFILE`
- `DIFFPAL_PROVIDER`
- `DIFFPAL_POLICY`
- `DIFFPAL_BLOCK_ON`
- `DIFFPAL_OPENAI_MODEL`
- `DIFFPAL_REVIEW_MAX_FILES`
- `DIFFPAL_REVIEW_CONTEXT_LINES`
- `DIFFPAL_REVIEW_LANGUAGE`
- `DIFFPAL_REVIEW_CHECKS`

## Exit Codes

| Code | Meaning |
| --- | --- |
| `0` | Review completed. |
| `1` | Blocking findings exist and `--gate` was set. |
| `2` | Config, profile, provider, or auth validation failed. |
| `3` | Provider timeout, rate limit, or transient failure. |
| `4` | Platform publish failed. |
| `5` | Internal tooling failure. |
| `130` | Interrupted or cancelled. |
