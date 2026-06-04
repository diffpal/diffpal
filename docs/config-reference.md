# DiffPal Configuration Reference

## File Layout and Precedence

Configuration is loaded from one selected file:

1. `--config-dir/diffpal/config.yaml`
2. `--config-dir/config.yaml`
3. `.config/diffpal/config.yaml` in the repository

Then DiffPal applies profile overlay, environment overrides, and command flags.
Profile selection follows `--profile`, then `DIFFPAL_PROFILE`, then implicit
`default`.

Required top-level fields:

- `version`: must be `v1`
- `defaults.provider`: active provider key
- `defaults.policy`: active policy key, defaults to `default`
- `providers`: Norma runtime provider registry
- `policies.<name>.block_on`: blocking threshold (`low|medium|high|critical`)
- `review`: review defaults

## Full Example

```yaml
version: v1

defaults:
  provider: openai-fast
  policy: default

providers:
  openai-fast:
    type: openai
    openai:
      model: "${DIFFPAL_OPENAI_MODEL}"
      api_key: "${OPENAI_API_KEY}"
  copilot-acp:
    type: copilot_acp
    copilot_acp:
      mode: ""

policies:
  default:
    block_on: high

review:
  context_lines: 20
  max_files: 200
  chunking:
    max_patch_chars: 12000
    max_files_per_chunk: 20

platforms:
  github:
    auth:
      token: "${GITHUB_TOKEN}"
  gitlab:
    auth:
      job_token: "${CI_JOB_TOKEN}"
      api_token: "${GITLAB_TOKEN}"
  azure:
    auth:
      system_access_token: "${SYSTEM_ACCESSTOKEN}"
      pat: "${AZURE_DEVOPS_EXT_PAT}"

profiles:
  copilot-acp:
    defaults:
      provider: copilot-acp
    policies:
      default:
        block_on: critical
```

## Envsubst and Overrides

Config files support envsubst-style placeholders before YAML parsing:

- `$VAR`
- `${VAR}`

Missing referenced variables fail config load. Quote substituted values when they
may contain YAML-significant characters:

```yaml
api_key: "${OPENAI_API_KEY}"
token: "${GITHUB_TOKEN}"
```

Environment overrides:

- `DIFFPAL_PROFILE`
- `DIFFPAL_PROVIDER`
- `DIFFPAL_POLICY`
- `DIFFPAL_BLOCK_ON`
- `DIFFPAL_REVIEW_MAX_FILES`
- `DIFFPAL_REVIEW_CONTEXT_LINES`

## Platform Auth

Host review modes resolve platform API credentials from direct config values.
Use envsubst placeholders to inject secrets at runtime:

- `platforms.github.auth.token`
- `platforms.gitlab.auth.api_token`
- `platforms.gitlab.auth.job_token`
- `platforms.azure.auth.system_access_token`
- `platforms.azure.auth.pat`

Rules:

- `review local` ignores `platforms`.
- `review github` requires `token`.
- `review gitlab` prefers `api_token`, then falls back to `job_token`.
- `review ado` uses `platforms.azure` and prefers `system_access_token`, then falls back to `pat`.

## Policy and Exit Codes

`block_on` marks findings at or above a severity threshold as blocking. It does
not mean the tooling failed.

- `0`: review completed; non-gated runs may still report `status=blocked`
- `1`: review blocked because `--gate` was set and blocking findings exist
- `2`: config/profile/provider/auth validation failure
- `3`: provider temporary failure (timeout/rate-limit/network)
- `4`: publish failure
- `5`: internal unexpected tooling failure
- `130`: interrupted / cancelled

DiffPal normalizes findings into canonical `findings.json` and derives
deterministic IDs using repository, review, sha, path, line range, rule, and
evidence/message signatures.
