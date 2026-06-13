# Azure DevOps Adapter Contract (v1)

Public CLI naming uses `ado`; config uses `azure`:

- command: `diffpal review ado`
- config: `diffpal.platforms.azure`

For a copy-paste Azure Pipelines setup, start with the
[CI setup guide](ci-examples.md#azure-pipelines). This page documents adapter
behavior and task requirements.

## Context resolution

`Azure` context is resolved from:

1. Explicit command args (`--base`, `--head`)
   Pull request identity is resolved from pipeline metadata and optional payload data.
2. Pipeline variables:
   - `SYSTEM_PULLREQUEST_PULLREQUESTID`
   - `SYSTEM_PULLREQUEST_SOURCECOMMITID`
   - `SYSTEM_PULLREQUEST_TARGETCOMMITID`
   - `SYSTEM_PULLREQUEST_SOURCEBRANCH`
   - `SYSTEM_PULLREQUEST_TARGETBRANCH`
   - `BUILD_REPOSITORY_ID`
   - `SYSTEM_COLLECTIONURI`
3. Optional payload path (`SYSTEM_PULLREQUEST_EVENT_PAYLOAD`)

Required:

- pull request ID
- head SHA
- base SHA
- repository/project context
- token source:
  - `system_access_token`
  - `pat`

## PR thread publishing

- Only actionable findings with canonical `start_line > 0` and high confidence produce inline thread actions.
- Key model:
  - `path + ":" + start_line + ":" + end_line + ":" + ruleID`
- Re-runs are idempotent via stored key + finding ID:
  - same key + same finding ID → skip
  - same key + different finding ID → update
- Thread plans also carry the PR comparison pair (`base_sha`, `head_sha`) used to map comments to the reviewed change range.

## Status mapping

- `succeeded`: no blocking findings
- `pending`: findings exist but no merge blockers
- `failed`: blocking findings or tooling error

Status payload name should be stable and branch-policy-compatible, e.g.:

- `DiffPal Review`

## Token and setup guidance

- The `DiffPalReview@1` task installs `@diffpal/diffpal` by default. Set
  `diffpalVersion` to pin a version or dist-tag.
- Install the provider command separately, for example
  `npm install --global @openai/codex@latest @normahq/codex-acp-bridge@latest`.
- Set `install: false` to use `diffpal` from `PATH`, or set `diffpalPath` to a
  custom binary path. Custom paths skip automatic installation.
- Optional task inputs `language`, `reviewChecks`, and `feedback` map to the
  CLI flags `--language`, `--review-checks`, and `--feedback`.
- `feedback: balanced` is the default and publishes status, a PR summary
  thread, and actionable high-confidence inline threads.
- Config auth values:
  - `diffpal.platforms.azure.auth.system_access_token`
  - `diffpal.platforms.azure.auth.pat`
- Standard CI env fallbacks are `SYSTEM_ACCESSTOKEN` and `AZURE_DEVOPS_EXT_PAT`.
- Use `SYSTEM_ACCESSTOKEN` for pipeline-scoped access.
- Prefer the standard CI environment fallback for `SYSTEM_ACCESSTOKEN` rather than committed token placeholders. If you use envsubst placeholders for explicit config injection, define those variables in the pipeline before loading config.
- Azure Pipelines must enable `Allow scripts to access the OAuth token` so `SYSTEM_ACCESSTOKEN` is present.
- Keep token scope to PR validation service connections or project defaults.
- Avoid broad service permissions in non-interactive PR contexts.
- A typical rerun flow is: `review ado` recomputes the findings bundle, then `summary`, `threads`, and `status` reconcile against the same PR/base/head pair instead of creating duplicate thread keys.
