# Configuration Reference

DiffPal reads repository configuration from `.config/diffpal/config.yaml`.
The file is YAML, uses top-level `version: v1`, and keeps review policy with
the repository.

Task-oriented setup lives in [Getting Started](../getting-started/README.md),
[Providers](../providers/README.md), and [Integrations](../integrations/README.md).

## Loading And Overrides

Default config root:

- `.config/diffpal/config.yaml`

When `--config-dir <root>` is set, DiffPal searches:

1. `<root>/diffpal/config.yaml`
2. `<root>/config.yaml`
3. `.config/diffpal/config.yaml`

Profiles are selected by `--profile` or `DIFFPAL_PROFILE`. Environment
overrides are applied after config loading and before validation.

## Complete Hierarchy

```yaml
version: v1

runtime:
  providers:
    <provider-id>:
      type: <provider-type>
      mcp_servers: []
      system_instructions: ""
      <provider-type>: {}
  mcp_servers:
    <server-id>:
      type: stdio

diffpal:
  provider: <provider-id>
  gate:
    block_on: high
  review:
    language: en
    instructions: ""
    timeout: 5m
  platforms:
    github:
      auth:
        token: ""
    gitlab:
      auth:
        job_token: ""
        api_token: ""
    azure:
      auth:
        system_access_token: ""
        pat: ""

profiles:
  ci:
    runtime: {}
    diffpal: {}
```

## Required Fields

| Field | Required | Default | Allowed values |
| --- | --- | --- | --- |
| `version` | Optional | `v1` from defaults | empty or `v1` |
| `runtime.providers` | Required for review | empty | map of provider IDs |
| `runtime.providers.<id>.type` | Required | none | `generic_acp`, `gemini_acp`, `codex_acp`, `opencode_acp`, `copilot_acp`, `claude_code_acp`, `openai`, `aistudio`, `pool` |
| `diffpal.provider` | Required | empty | provider ID present in `runtime.providers` |
| `diffpal.gate.block_on` | Optional | `high` | `low`, `medium`, `high`, `critical` |
| `diffpal.review.language` | Optional | `en` | any single-line language value |
| `diffpal.review.instructions` | Optional | empty | string |
| `diffpal.review.timeout` | Optional | `5m` | positive Go duration such as `180s`, `5m`, `10m` |

`diffpal.provider` must match one key in `runtime.providers`.

## Runtime Providers

Each provider is discriminated by `type`; the matching block contains
type-specific settings.

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

Common provider fields:

| Field | Applies to | Purpose |
| --- | --- | --- |
| `mcp_servers` | all providers | IDs from `runtime.mcp_servers` attached to this provider. |
| `system_instructions` | all providers | Provider-level instructions applied by the runtime. |
| `generic_acp`, `gemini_acp`, `codex_acp`, `opencode_acp`, `copilot_acp`, `claude_code_acp` | ACP providers | ACP runtime block. |
| `openai`, `aistudio` | hosted API providers | Hosted API runtime block. |
| `pool` | pool provider | Ordered failover member list. |

ACP runtime block fields:

| Field | Required | Default | Allowed values |
| --- | --- | --- | --- |
| `cmd` | Optional | provider alias default, or required for custom `generic_acp` use | string list |
| `extra_args` | Optional | empty | string list |
| `model` | Optional | provider-specific | non-blank string |
| `reasoning_effort` | Optional | provider-specific | `minimal`, `low`, `medium`, `high`, `xhigh` |
| `mode` | Optional | provider-specific | non-blank string |

Hosted API block fields:

| Field | Required | Default | Notes |
| --- | --- | --- | --- |
| `api_key` | Required for hosted review unless supplied by env | empty | `OPENAI_API_KEY` for `openai`, `GEMINI_API_KEY` for `aistudio`. |
| `model` | Required for hosted review | empty | Can be overridden for `openai` by `DIFFPAL_OPENAI_MODEL`. |

Pool block fields:

| Field | Required | Default | Notes |
| --- | --- | --- | --- |
| `members` | Optional | empty | Provider IDs in ordered failover order. |

Provider installation and authentication are documented in
[Providers](../providers/README.md).

## MCP Servers

MCP servers are declared once and referenced by provider ID:

```yaml
runtime:
  mcp_servers:
    repo-docs:
      type: stdio
      cmd: ["your-docs-mcp-server"]
      args: ["--root", "."]
      env:
        DOCS_TOKEN: "${DOCS_TOKEN}"
  providers:
    codex-acp:
      type: codex_acp
      mcp_servers: [repo-docs]
      codex_acp:
        reasoning_effort: low
```

MCP server fields:

| Field | Required | Applies to | Purpose |
| --- | --- | --- | --- |
| `type` | Required | all | `stdio`, `http`, or `sse`. |
| `cmd` | Optional | `stdio` | Executable argv prefix. |
| `args` | Optional | `stdio` | Arguments appended after `cmd`. |
| `env` | Optional | `stdio` | Environment variables for the server process. |
| `working_dir` | Optional | `stdio` | Server working directory. |
| `url` | Optional | `http`, `sse` | Server endpoint. |
| `headers` | Optional | `http`, `sse` | Request headers. |

Keep MCP credentials in CI secrets.

## Review Settings

| Field | Default | Notes |
| --- | --- | --- |
| `diffpal.review.language` | `en` | Used for finding text and summaries. Must be one line. |
| `diffpal.review.instructions` | empty | Appended to the review prompt as repository-owned tuning. |
| `diffpal.review.timeout` | `5m` | Per-attempt provider review timeout. |

`--language`, `--instructions`, and `--instructions-file` can override or
append instructions for one CLI run. See [CLI](cli.md).

## Review Contract And Severity Matrix

DiffPal asks providers to report only discrete issues the pull request author
would likely fix before merging, only when the issue is introduced or made
worse by the patch. Providers must use the DiffPal finding taxonomy, prefer
high signal over high recall, use the smallest useful changed-line range, and
may return `review_result` when they can summarize the outcome.

Severity is based on concrete impact, not confidence or preference.

- critical: changed code can directly cause severe compromise, destructive data loss, privilege bypass, total outage, or unrecoverable corruption.
- high: changed code can cause an exploitable security flaw, user-visible data corruption, frequent crash, major outage risk, or severe performance regression on a common path.
- medium: changed code can cause an edge-case correctness failure, intermittent reliability issue, meaningful performance cost, confusing maintainability risk, or missing coverage for important behavior.
- low: changed code has a localized maintainability, testing, or style issue with clear evidence and a concrete improvement, but no immediate user-visible failure.
- security: use critical for direct compromise, credential exposure, destructive access, or privilege bypass; high for exploitable vulnerabilities with meaningful impact; medium for plausible weaknesses requiring extra conditions; low for hardening gaps with limited direct exploitability.
- correctness: use critical for unrecoverable corruption or system-wide wrong behavior; high for common-path wrong results or data corruption; medium for plausible edge-case failures; low for small inconsistencies with bounded impact.
- reliability: use critical for total outage, deadlock, or unrecoverable resource exhaustion; high for frequent crashes, leaks, or retry storms; medium for intermittent failure modes; low for localized resilience or observability gaps.
- performance: use critical for regressions that can make the service unavailable or explode cost; high for severe common-path latency, memory, or query regressions; medium for measurable inefficient behavior; low for small avoidable inefficiencies with clear evidence.
- maintainability: use critical only when the change creates an immediate operational hazard; high when the change makes future safe modification very risky; medium for confusing structure likely to cause defects; low for localized clarity or consistency issues.
- testing: use critical only when missing validation can allow severe unsafe behavior to ship; high for untested high-risk behavior; medium for missing meaningful coverage of new behavior; low for small missing edge-case coverage.
- style: use critical, high, or medium only when the issue has a non-style impact and should usually be reclassified; use low for repo-enforced style/readability issues that are actionable.

## Gate

```yaml
diffpal:
  gate:
    block_on: high
```

`block_on` sets the minimum severity that counts as blocking. The allowed values
are `low`, `medium`, `high`, and `critical`. Use `--gate` in CI to make
blocking findings fail the process. See [Exit behavior](exit-behavior.md).

## Platform Auth

Platform auth can live in config, but standard CI environment variables are
preferred.

| Platform | Config field | Preferred env |
| --- | --- | --- |
| GitHub | `diffpal.platforms.github.auth.token` | `GITHUB_TOKEN` |
| GitLab | `diffpal.platforms.gitlab.auth.job_token` | `CI_JOB_TOKEN` |
| GitLab | `diffpal.platforms.gitlab.auth.api_token` | `GITLAB_TOKEN` |
| Azure DevOps | `diffpal.platforms.azure.auth.system_access_token` | `SYSTEM_ACCESSTOKEN` |
| Azure DevOps | `diffpal.platforms.azure.auth.pat` | `AZURE_DEVOPS_EXT_PAT` |

Blank auth values fail validation. Missing optional config values are allowed
when the matching environment variable will be present in CI.

## Profiles

Profiles override the same root sections:

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

| Variable | Overrides |
| --- | --- |
| `DIFFPAL_PROFILE` | selected profile when `--profile` is not set |
| `DIFFPAL_PROVIDER` | `diffpal.provider` |
| `DIFFPAL_BLOCK_ON` | `diffpal.gate.block_on` |
| `DIFFPAL_OPENAI_MODEL` | `runtime.providers.*.openai.model` |
| `DIFFPAL_REVIEW_LANGUAGE` | `diffpal.review.language` |
| `DIFFPAL_REVIEW_INSTRUCTIONS` | `diffpal.review.instructions` |
| `DIFFPAL_REVIEW_TIMEOUT` | `diffpal.review.timeout` |

Config loading also expands `$VAR` and `${VAR}` placeholders before YAML
parsing. Missing placeholder variables fail config loading.
