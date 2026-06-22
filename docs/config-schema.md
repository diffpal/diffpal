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
  platforms:
    github: {}
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
- `DIFFPAL_REVIEW_INSTRUCTIONS`

Config files expand `$VAR` and `${VAR}` before YAML parsing when placeholders
are used for required values. Missing referenced variables fail config load, so
do not use placeholders for optional CI credentials; omit those fields and let
runtime auth resolution read standard CI environment variables. Quote
substituted values.

Platform auth can be supplied either by config fields or standard CI
environment variables: `GITHUB_TOKEN`, `GITLAB_TOKEN`, `CI_JOB_TOKEN`,
`SYSTEM_ACCESSTOKEN`, and `AZURE_DEVOPS_EXT_PAT`.

`runtime.providers` is the provider-choice boundary. DiffPal can use any
ACP-compatible CLI through `generic_acp`:

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

Alias provider types are available for common ACP CLIs:
`codex_acp`, `copilot_acp`, `gemini_acp`, `claude_code_acp`, and
`opencode_acp`. Hosted provider types include `openai`, `aistudio`, and
`pool`.

The default public onboarding recipe selects `codex-acp`. Install it with
`npm install --global @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3`
and authenticate Codex with `OPENAI_API_KEY` in CI.

For GitHub, DiffPal always publishes a pull request review. Findings are
published as file-level review comments.

Validation requires `version: v1`, a `diffpal.provider` key present in
`runtime.providers`, and a valid `diffpal.gate.block_on` severity. Changing
providers means changing the selected `runtime.providers` entry plus the
matching CI install/auth steps.

`diffpal.review.language` defaults to `en` and can be overridden by
`--language`. Use `diffpal.review.instructions`, `--instructions`, or
`--instructions-file` for repository-local prompt tuning and scope extensions,
for example OWASP-focused security review.
