# GitLab CI

Use this page to run DiffPal in GitLab merge request pipelines.

## Supported Outputs

- Merge request summary discussion.
- File-level discussions for actionable findings.
- Code Quality report.
- SARIF report.
- Commit status named `DiffPal Review`.

## Prerequisites

- A GitLab project with merge request pipelines.
- A committed DiffPal config at `.config/diffpal/config.yaml`.
- A provider secret such as `OPENAI_API_KEY`.
- A platform token path that can publish MR feedback.

See [Shared Setup](README.md#shared-setup) and
[Providers](../providers/README.md).

## Required Checkout Behavior

Set full git depth so DiffPal can compare base and head:

```yaml
variables:
  GIT_DEPTH: "0"
```

Pass `--base "$CI_MERGE_REQUEST_DIFF_BASE_SHA"` and
`--head "$CI_COMMIT_SHA"` when running `diffpal review gitlab`.

## Required Token And Minimum Permissions

DiffPal can publish with:

| Token | Use when |
| --- | --- |
| `CI_JOB_TOKEN` | Your instance allows MR API publishing with the built-in job token. |
| `GITLAB_TOKEN` | You need a dedicated API token for MR discussions/status publishing. |

Use protected/masked variables for provider tokens. Prefer the least-privilege
token that can create MR discussions, publish commit status, and upload reports.
Keep host tokens separate from provider credentials such as `OPENAI_API_KEY`.

## Provider Installation And Authentication

Install and authenticate the selected provider before the DiffPal step. Use
[Providers](../providers/README.md) for Codex, Copilot, OpenCode, and custom
ACP-compatible CLI setup.

Provider credentials allow the selected third-party provider to process the
review input. Store them as protected and masked variables, and expose them only
to trusted pipelines. See
[Secrets and fork PRs](../guides/secrets-and-fork-prs.md).

## Minimal Pipeline

```yaml
stages:
  - review

diffpal-review:
  stage: review
  image: node:22
  rules:
    - if: '$CI_PIPELINE_SOURCE == "merge_request_event" && $CI_MERGE_REQUEST_SOURCE_PROJECT_PATH == $CI_PROJECT_PATH && $DIFFPAL_TRUSTED_REVIEW == "true"'
      when: manual
      allow_failure: false
    - when: never
  resource_group: "diffpal:$CI_MERGE_REQUEST_IID"
  before_script:
    - npm install --global @diffpal/diffpal@latest @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3
    - printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
  script:
    - >-
      diffpal --profile ci review gitlab
      --base "$CI_MERGE_REQUEST_DIFF_BASE_SHA"
      --head "$CI_COMMIT_SHA"
      --repo "$CI_PROJECT_PATH"
      --review-id "gitlab-mr-$CI_MERGE_REQUEST_IID"
      --feedback review
      --gate
  variables:
    GIT_DEPTH: "0"
  artifacts:
    when: always
    paths:
      - .artifacts/diffpal/
    reports:
      codequality: .artifacts/diffpal/codequality.json
      sarif: .artifacts/diffpal/diffpal.sarif
```

## Feedback Modes

Use `--feedback review` for a summary discussion plus file-level discussions.
Use `--feedback summary` for summary, status, Code Quality, and SARIF without
file-level discussions.

See [Feedback Modes](README.md#feedback-modes).

## Merge-Gate Setup

Pass `--gate`. Blocking findings return exit code `10` after publishing
succeeds and are represented by unresolved blocker discussions plus the
`DiffPal Review` status.

See [Merge Gates](README.md#merge-gates) and
[Exit behavior](../reference/exit-behavior.md).

## Fork Or Untrusted-Contribution Behavior

Keep provider credentials available only to trusted pipelines. For fork merge
requests, use no-secret CI or a maintainer-approved manual job. The minimal
pipeline and examples use same-project conditions, a manual
`DIFFPAL_TRUSTED_REVIEW` guard, and protected variables for secret-backed
review.

See [Secrets and fork PRs](../guides/secrets-and-fork-prs.md).

## Expected Results

- GitLab discussions for actionable findings when feedback is `review`.
- A merge request summary discussion.
- Code Quality and SARIF artifacts.
- A commit status named `DiffPal Review`.
- `.artifacts/diffpal/summary.md` in job artifacts.

## Common Failures

- `GIT_DEPTH: "0"` is missing.
- `CI_JOB_TOKEN` lacks MR API permissions; use `GITLAB_TOKEN`.
- Provider variables are not protected/masked or are unavailable to the job.
- The pipeline is not a merge request pipeline, so MR context is missing.

See [Common Failures](README.md#common-failures).

## Related Examples

- [Codex API key](../../examples/ci/gitlab/codex-api-key.yml)
- [Codex subscription auth](../../examples/ci/gitlab/codex-subscription.yml)
- [Copilot token](../../examples/ci/gitlab/copilot-github-token.yml)

Next step: use [Verify First Review](../getting-started/verify-first-review.md)
after the first GitLab pipeline completes.
