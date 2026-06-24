# GitLab CI

For a copy-paste GitLab CI setup, start with the
[integrations guide](README.md). This page documents setup requirements,
adapter behavior, and publishing semantics.

Examples:

- [Codex API key](../../examples/ci/gitlab/codex-api-key.yml)
- [Codex subscription auth](../../examples/ci/gitlab/codex-subscription.yml)
- [Copilot token](../../examples/ci/gitlab/copilot-github-token.yml)

Required variables:

| Name | Purpose |
| --- | --- |
| `CI_JOB_TOKEN` | Built-in token, when your instance allows MR API publishing. |
| `GITLAB_TOKEN` | Optional dedicated token when `CI_JOB_TOKEN` is not enough. |

Use protected/masked variables for provider tokens. If your project accepts fork
merge requests, keep provider tokens available only to trusted pipelines. The
examples restrict secret-backed review jobs to same-project merge requests with
`$CI_MERGE_REQUEST_SOURCE_PROJECT_PATH == $CI_PROJECT_PATH`.

What you should see:

- GitLab discussions for actionable findings.
- GitLab commit status named `DiffPal Review`.
- Code Quality and SARIF artifacts.
- `.artifacts/diffpal/summary.md` in job artifacts.
- Failed job when `--gate` is set and blocking findings exist.

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

Normal CI setup should use `--feedback`:

- `summary`: posts Code Quality/SARIF artifacts and one MR summary discussion, without file-level findings.
- `review`: publishes Code Quality/SARIF artifacts, one MR summary discussion, a commit status, and file-level discussions for every publishable finding.

Severity to discussion policy:

- `high/critical`: blocking discussion that remains unresolved until manual action
- `medium/low`: advisory file-level discussion that is resolved immediately

Each finding maps to a stable thread key:

- `path + ":" + start_line + ":" + category`

Re-publishing uses key + finding ID for idempotent update/skip.

## Code Quality and SARIF

- GitLab Code Quality artifact: `.artifacts/diffpal/codequality.json`
- SARIF artifact: `.artifacts/diffpal/diffpal.sarif`
- GitLab commit status artifact: `.artifacts/diffpal/gitlab-status.json`

Both artifacts are generated from one deterministic source so dedupe keys remain stable across re-runs.

## Gating

- Tool decision: `pass` (no findings), `warn` (advisory findings), `blocked` (blocking findings), `error` (tooling failure).
- Merge blocking is represented in:
  - unresolved discussions for blockers, and
  - a GitLab commit status named `DiffPal Review` / `diffpal/review`.
- `--gate` still controls the DiffPal process exit code: blocking findings return exit code `1` after publishing succeeds.

## Operational requirements

- Config auth values:
  - `diffpal.platforms.gitlab.auth.api_token`
  - `diffpal.platforms.gitlab.auth.job_token`
- Standard CI env fallbacks are `GITLAB_TOKEN` and `CI_JOB_TOKEN`; API token is preferred over job token.
- Envsubst placeholders such as `api_token: "${GITLAB_TOKEN}"` are supported when you want config-file injection, but missing referenced variables fail config load.
- Retry policy: platform retries are batched and use exponential backoff with idempotent thread keys.
