# DiffPal Config Schema

`config` is loaded from:

1. `--config-dir/diffpal/config.yaml`
2. `--config-dir/config.yaml`
3. `.config/diffpal/config.yaml`
4. profile overlay selected by `--profile` or `DIFFPAL_PROFILE`
5. environment overrides
6. CLI flags

Higher position in the list has higher precedence.

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
  ci:
    defaults:
      policy: default
```

Environment overrides:

- `DIFFPAL_PROVIDER`
- `DIFFPAL_PROFILE`
- `DIFFPAL_POLICY`
- `DIFFPAL_BLOCK_ON`
- `DIFFPAL_REVIEW_MAX_FILES`
- `DIFFPAL_REVIEW_CONTEXT_LINES`

Config files expand `$VAR` and `${VAR}` before YAML parsing. Missing variables
fail config load; quote substituted values.

Validation requires `version: v1`, a `defaults.provider` key present in
`providers`, a `defaults.policy` key present in `policies`, and a valid
`policies.<name>.block_on` severity.
