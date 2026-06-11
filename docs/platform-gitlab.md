# GitLab Adapter Contract (v1)

## Context resolution

`Review` and `publish` for GitLab resolve target PR context in this precedence:

1. Explicit command args supplied to GitLab context resolution
2. GitLab CI variables
3. GitLab event payload (`GITLAB_EVENT_PATH`/`CI_MERGE_REQUEST_EVENT_PATH`)

Required context:

- repository/project path
- merge request IID
- base SHA (`CI_MERGE_REQUEST_DIFF_BASE_SHA` or payload `object_attributes.oldrev`)
- head SHA (`CI_COMMIT_SHA` or payload `object_attributes.last_commit.id`)
- token mode:
  - `ci_job_token` for standard CI publishing
  - `gitlab_token` when a dedicated API token is configured

## Discussion publisher

Severity to discussion policy:

- `high/critical`: blocking discussion that remains unresolved until manual action
- `medium/low`: advisory summary stream instead of inline blocking discussion

Each finding maps to a stable thread key:

- `path + ":" + start_line + ":" + ruleID`

Re-publishing uses key + finding ID for idempotent update/skip.

## Code Quality and SARIF

- GitLab Code Quality artifact: `.artifacts/diffpal/codequality.json`
- SARIF artifact: `.artifacts/diffpal/diffpal.sarif`

Both artifacts are generated from one deterministic source so dedupe keys remain stable across re-runs.

## Gating

- Tool decision: `pass` (no findings), `warn` (advisory findings), `blocked` (blocking findings), `error` (tooling failure).
- Merge blocking is represented in:
  - unresolved discussions for blockers, and
  - optional blocked CI context when `--gate` is enabled.

## Operational requirements

- Config auth values:
  - `platforms.gitlab.auth.api_token`
  - `platforms.gitlab.auth.job_token`
- Standard CI env fallbacks are `GITLAB_TOKEN` and `CI_JOB_TOKEN`; API token is preferred over job token.
- Envsubst placeholders such as `api_token: "${GITLAB_TOKEN}"` are supported when you want config-file injection, but missing referenced variables fail config load.
- Retry policy: platform retries are batched and use exponential backoff with idempotent thread keys.
