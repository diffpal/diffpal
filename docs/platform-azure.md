# Azure DevOps Adapter Contract (v1)

Public CLI naming uses `ado`; config uses `azure`:

- command: `diffpal review ado`
- config: `platforms.azure`

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
  - `none`

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

- The `DiffPalReview@1` task expects the `diffpal` CLI to be installed before
  the task runs. Pin the CLI to an audited SemVer version, for example
  `npm install @diffpal/diffpal@1.2.3`, then `diffpalPath:
  ./node_modules/.bin/diffpal`.
- Config auth values:
  - `platforms.azure.auth.system_access_token`
  - `platforms.azure.auth.pat`
- Standard CI env fallbacks are `SYSTEM_ACCESSTOKEN` and `AZURE_DEVOPS_EXT_PAT`.
- Use `SYSTEM_ACCESSTOKEN` for pipeline-scoped access.
- Prefer the standard CI environment fallback for `SYSTEM_ACCESSTOKEN` rather than committed token placeholders. If you use envsubst placeholders for explicit config injection, define those variables in the pipeline before loading config.
- Azure Pipelines must enable `Allow scripts to access the OAuth token` so `SYSTEM_ACCESSTOKEN` is present.
- Keep token scope to PR validation service connections or project defaults.
- Avoid broad service permissions in non-interactive PR contexts.
- A typical rerun flow is: `review ado` recomputes the findings bundle, then `threads` and `status` reconcile against the same PR/base/head pair instead of creating duplicate thread keys.
