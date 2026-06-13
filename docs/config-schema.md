# DiffPal Config Schema

`config` is resolved in this order, from base file to highest-priority override:

1. `--config-dir/diffpal/config.yaml`
2. `--config-dir/config.yaml`
3. `.config/diffpal/config.yaml`
4. profile overlay selected by `--profile` or `DIFFPAL_PROFILE`
5. environment overrides
6. CLI flags

Later entries override earlier entries. Profile overlays live under
`profiles.<name>.diffpal` and `profiles.<name>.runtime`.

```yaml
version: v1

runtime:
  providers:
    codex-acp:
      type: codex_acp
      codex_acp:
        reasoning_effort: low
  mcp_servers: {}

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

profiles:
  ci:
    diffpal:
      gate:
        block_on: high
```

Environment overrides:

- `DIFFPAL_PROVIDER`
- `DIFFPAL_PROFILE`
- `DIFFPAL_BLOCK_ON`
- `DIFFPAL_OPENAI_MODEL`
- `DIFFPAL_REVIEW_LANGUAGE`
- `DIFFPAL_REVIEW_CHECKS`
- `DIFFPAL_REVIEW_INSTRUCTIONS`

Config files expand `$VAR` and `${VAR}` before YAML parsing when placeholders
are used for required values. Missing referenced variables fail config load, so
do not use placeholders for optional CI credentials; omit those fields and let
runtime auth resolution read standard CI environment variables. Quote
substituted values.

Platform auth can be supplied either by config fields or standard CI
environment variables: `GITHUB_TOKEN`, `GITLAB_TOKEN`, `CI_JOB_TOKEN`,
`SYSTEM_ACCESSTOKEN`, and `AZURE_DEVOPS_EXT_PAT`.

The default public onboarding provider is `codex-acp`. Install it with
`npm install --global @openai/codex@latest @normahq/codex-acp-bridge@latest`
and authenticate Codex with `OPENAI_API_KEY` in CI.

`diffpal.platforms.github.summary_comment.enabled` defaults to `true`. When
`summary` mode is selected, DiffPal posts or updates a PR-level GitHub summary
comment even if there are no findings.

Validation requires `version: v1`, a `diffpal.provider` key present in
`runtime.providers`, and a valid `diffpal.gate.block_on` severity.

`diffpal.review.language` defaults to `en`. `diffpal.review.checks` defaults to
`security`, `bugs`, `performance`, and `best-practices`; those values can be
overridden by the `--language` and `--review-checks` review flags. Use
`diffpal.review.instructions`, `--instructions`, or `--instructions-file` for
repository-local prompt tuning.
