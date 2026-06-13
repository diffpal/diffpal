# DiffPal Config Reference

DiffPal reads one repository config file:

`.config/diffpal/config.yaml`

Generate a starter file with:

```bash
diffpal init
```

## Default Codex Config

The public onboarding path uses Codex ACP:

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
    instructions: |
      Prefer actionable findings that are directly supported by the diff.
    checks:
      - security
      - bugs
      - performance
      - best-practices
  platforms:
    github:
      summary_comment:
        enabled: true
    gitlab: {}
    azure: {}
```

Install the matching provider command in CI:

```bash
npm install --global @openai/codex@latest
```

Set `OPENAI_API_KEY` as a CI secret and authenticate Codex with
`codex login --with-api-key`. Do not commit token values into the
config file.

## Root Sections

| Field | Purpose |
| --- | --- |
| `version` | Config schema version. Use `v1`. |
| `runtime.providers` | Norma runtime provider definitions. |
| `runtime.mcp_servers` | Optional MCP servers shared by providers. |
| `diffpal.provider` | Provider ID selected for reviews. Must exist in `runtime.providers`. |
| `diffpal.gate.block_on` | Minimum severity that marks a finding as blocking. |
| `diffpal.review` | User-facing review language and check scopes. |
| `diffpal.platforms` | Optional platform publishing configuration. |
| `profiles.<name>.diffpal` | Profile-specific DiffPal overrides. |
| `profiles.<name>.runtime` | Profile-specific runtime overrides. |

## Review Settings

| Field | Default | Purpose |
| --- | --- | --- |
| `diffpal.review.language` | `en` | Language for finding text and summaries. |
| `diffpal.review.instructions` | empty | Optional repository-local prompt tuning appended to the review instruction. |
| `diffpal.review.checks` | `security`, `bugs`, `performance`, `best-practices` | Review scopes to request from the provider. |

Review checks map to finding categories:

| Check | Categories |
| --- | --- |
| `security` | security |
| `bugs` | correctness, reliability |
| `performance` | performance |
| `best-practices` | maintainability, testing, style |

Override review settings per run:

```bash
diffpal review github \
  --base "$BASE_SHA" \
  --head "$HEAD_SHA" \
  --language en \
  --review-checks security,bugs,performance,best-practices \
  --instructions-file .config/diffpal/review-instructions.md \
  --feedback balanced
```

## Gate

`diffpal.gate.block_on` controls which findings are blocking:

```yaml
diffpal:
  gate:
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
| GitHub | `GITHUB_TOKEN` | `diffpal.platforms.github.auth.token` |
| GitLab | `CI_JOB_TOKEN` or `GITLAB_TOKEN` | `diffpal.platforms.gitlab.auth.job_token`, `diffpal.platforms.gitlab.auth.api_token` |
| Azure | `SYSTEM_ACCESSTOKEN` | `diffpal.platforms.azure.auth.system_access_token` |

Only use envsubst placeholders for values that are guaranteed to exist:

```yaml
diffpal:
  platforms:
    github:
      auth:
        token: "${GITHUB_TOKEN}"
```

Missing envsubst variables fail config loading. For optional CI credentials,
omit the config value and let DiffPal read the standard environment variable.

## Alternate Hosted OpenAI Provider

Codex ACP is the default onboarding provider. If you prefer hosted OpenAI,
switch the selected provider and add a matching runtime provider:

```yaml
runtime:
  providers:
    openai-fast:
      type: openai
      openai:
        model: "${DIFFPAL_OPENAI_MODEL}"
        api_key: "${OPENAI_API_KEY}"

diffpal:
  provider: openai-fast
```

Then set `OPENAI_API_KEY` in CI.

## Profiles

Profiles override the base document under the same root sections:

```yaml
profiles:
  ci:
    diffpal:
      gate:
        block_on: high
      review:
        language: en
```

Select a profile with `--profile ci` or `DIFFPAL_PROFILE=ci`.

## Environment Overrides

These environment variables override config values:

- `DIFFPAL_PROFILE`
- `DIFFPAL_PROVIDER`
- `DIFFPAL_BLOCK_ON`
- `DIFFPAL_OPENAI_MODEL`
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
