# DiffPal Config Reference

DiffPal reads one repository config file:

`.config/diffpal/config.yaml`

Generate a first-run onboarding config with:

```bash
diffpal init --wizard --setup codex-api-key --platform github
```

Use plain `diffpal init` only when you want the older starter workspace plus
template snippets. The wizard path writes a complete recipe config with a
visible profile, for example `profiles.ci`.

## Provider Model

DiffPal's provider-agnostic boundary is `runtime.providers`. Each entry
describes one AI provider, ACP-compatible CLI, hosted API provider, or provider
pool. `diffpal.provider` selects the entry used for reviews.

Use `generic_acp` for any CLI that can start an ACP stdio server:

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

Install and authenticate that CLI in CI before running DiffPal. DiffPal sends
the structured diff review request and owns the review output contract; the ACP
agent owns its model, tools, account, and provider-specific credentials.

Supported runtime provider types include:

- `generic_acp`
- `codex_acp`
- `copilot_acp`
- `gemini_acp`
- `claude_code_acp`
- `opencode_acp`
- `openai`
- `aistudio`
- `pool`

## Default Codex Recipe

The default public onboarding recipe uses Codex ACP:

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

profiles:
  ci:
    diffpal:
      gate:
        block_on: high
```

Install the matching provider command in CI:

```bash
npm install --global @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3
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

## Prompt Pack

DiffPal review prompts are versioned as Prompt Pack v1. Generated findings
artifacts include prompt metadata so review output can be traced back to the
prompt contract:

- `prompt_id`: `diffpal.review`
- `prompt_version`: `v1.2.2`
- `purpose`: `review_changed_diff`
- `schema_version`: `findings.v2`

`diffpal.review.instructions`, `--instructions`, and `--instructions-file`
are appended as repository-local tuning in a dedicated prompt section. DiffPal
sends a compact review task snapshot instead of preloading every patch or file
snippet. Providers inspect the requested base/head diff through their available
Git and filesystem tooling.

Prompt Pack v1.2.2 labels commit messages, diffs, tool output, code, comments,
docs, test fixtures, and file contents as untrusted review evidence, never as
role changes or instructions to follow. It also follows a high-signal review
threshold: report only patch-introduced issues the pull request author would
likely fix before merging.

Use the offline debug harness to inspect the active prompt and task snapshot
without spending provider quota:

```bash
diffpal debug prompt --base origin/main --head HEAD --profile ci --format text
```

The command still loads config, collects the git diff, runs review
normalization, and writes a schema-valid mock findings bundle. It replaces only
the provider call with a local debug runtime.

Review checks map to finding categories:

| Check | Categories |
| --- | --- |
| `security` | security |
| `bugs` | correctness, reliability |
| `performance` | performance |
| `best-practices` | maintainability, testing, style |

## Severity Matrix

DiffPal asks providers to classify severity by concrete impact, not confidence
or preference. The active Prompt Pack uses the same definitions:

- Severity is based on concrete impact, not confidence or preference.
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

## MCP Servers

MCP servers are declared once under `runtime.mcp_servers`, then attached to the
provider that should see them. Use this when the review agent needs controlled
access to repository tools, internal docs, ticket search, or another local
capability.

```yaml
runtime:
  mcp_servers:
    repo-docs:
      type: stdio
      cmd: ["your-docs-mcp-server"]
      args: ["--root", "."]
      env:
        DOCS_TOKEN: "${DOCS_TOKEN}"
    policy-api:
      type: http
      url: "https://policy.example.com/mcp"
      headers:
        Authorization: "Bearer ${POLICY_MCP_TOKEN}"
  providers:
    codex-acp:
      type: codex_acp
      mcp_servers:
        - repo-docs
        - policy-api
      codex_acp:
        reasoning_effort: low

diffpal:
  provider: codex-acp
```

Keep MCP credentials in CI secrets. As with platform tokens, quote envsubst
placeholders and avoid optional placeholders unless the variable always exists
in that job.

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
    openai-api:
      type: openai
      openai:
        model: "${DIFFPAL_OPENAI_MODEL}"
        api_key: "${OPENAI_API_KEY}"

diffpal:
  provider: openai-api
```

Then set `OPENAI_API_KEY` in CI.

DiffPal passes base/head review metadata to the provider. The provider inspects
the diff and supporting code through its available Git and filesystem tooling;
DiffPal keeps provider choice at the `runtime.providers` boundary and validates
returned findings against the changed ranges it collected internally.

## Provider Auth Recipes

Copy-paste examples are in [`../examples`](../examples/README.md). The config
shape stays the same across CI systems; only the provider install/auth step
changes. These recipes are maintained examples; the product boundary for
provider choice is the `runtime.providers` entry selected by `diffpal.provider`.

### Generic ACP CLI

Use [`examples/configs/generic-acp/config.yaml`](../examples/configs/generic-acp/config.yaml).

Replace `generic_acp.cmd` with the command that starts your provider's ACP
stdio server:

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

Required secret: provider-specific.

### Codex API Key

Use [`examples/configs/codex-api-key/config.yaml`](../examples/configs/codex-api-key/config.yaml).

Install and authenticate the provider in CI:

```bash
npm install --global @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3
printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
```

Required secret: `OPENAI_API_KEY`.

### Codex Subscription Auth

Use [`examples/configs/codex-subscription/config.yaml`](../examples/configs/codex-subscription/config.yaml).

Generate a fresh `CODEX_AUTH_JSON_B64` value with the command recipe in
[`examples/README.md`](../examples/README.md#generate-codex_auth_json_b64).

Restore an existing Codex auth file in trusted CI:

```bash
mkdir -p "$HOME/.codex"
printf '%s' "$CODEX_AUTH_JSON_B64" | base64 --decode > "$HOME/.codex/auth.json"
chmod 600 "$HOME/.codex/auth.json"
```

Required secret: `CODEX_AUTH_JSON_B64`.

Use this only in trusted same-repository CI. Do not expose a restored Codex auth
file to untrusted fork pull requests or merge requests.

### Copilot Fine-Grained PAT

Use [`examples/configs/copilot-github-token/config.yaml`](../examples/configs/copilot-github-token/config.yaml).

Install the provider in CI:

```bash
npm install --global @github/copilot@1.0.61
```

Required secret: `COPILOT_GITHUB_TOKEN`.

The Copilot CLI reads `COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, then `GITHUB_TOKEN`.
Use `COPILOT_GITHUB_TOKEN` for DiffPal so the provider token is separate from
the platform publish token. The token must be a fine-grained GitHub PAT v2 with
the Copilot Requests permission; classic PATs are not supported by the Copilot
CLI.

### OpenCode ACP

Use [`examples/configs/opencode-acp/config.yaml`](../examples/configs/opencode-acp/config.yaml).

DiffPal's runtime starts OpenCode with its ACP command:

```yaml
runtime:
  providers:
    opencode-acp:
      type: opencode_acp
      opencode_acp:
        model: opencode/big-pickle

diffpal:
  provider: opencode-acp
```

Install and authenticate `opencode` in CI before running DiffPal. If you need a
custom OpenCode mode or model, add supported `opencode_acp` fields such as
`mode`, `model`, or `extra_args`.

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
