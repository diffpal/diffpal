# DiffPal Config Schema

`config` is resolved in this order, from base file to highest-priority override:

1. `--config-dir/diffpal/config.yaml`
2. `--config-dir/config.yaml`
3. `.config/diffpal/config.yaml`
4. profile overlay selected by `--profile` or `DIFFPAL_PROFILE`
5. environment overrides
6. CLI flags

Later entries override earlier entries; CLI flags have the highest precedence.

```yaml
version: v1

defaults:
  provider: openai-fast
  policy: default

providers:
  openai-fast:
    type: openai
    openai:
      model: gpt-5-mini

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
    summary_comment:
      enabled: true
  gitlab: {}
  azure: {}

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
- `DIFFPAL_OPENAI_MODEL`
- `DIFFPAL_REVIEW_MAX_FILES`
- `DIFFPAL_REVIEW_CONTEXT_LINES`

Config files expand `$VAR` and `${VAR}` before YAML parsing when placeholders
are used for required values. Missing referenced variables fail config load, so
do not use placeholders for optional CI credentials; omit those fields and let
runtime auth resolution read standard CI environment variables. Quote
substituted values.

Platform auth can be supplied either by config fields or standard CI
environment variables: `GITHUB_TOKEN`, `GITLAB_TOKEN`, `CI_JOB_TOKEN`,
`SYSTEM_ACCESSTOKEN`, and `AZURE_DEVOPS_EXT_PAT`.

`platforms.github.summary_comment.enabled` defaults to `true`. When `summary`
mode is selected, DiffPal posts or updates a PR-level GitHub summary comment
even if there are no findings.

Validation requires `version: v1`, a `defaults.provider` key present in
`providers`, a `defaults.policy` key present in `policies`, and a valid
`policies.<name>.block_on` severity.
