# Verify First Review

Use this page to confirm that the first DiffPal run produced a complete review,
even when it found no issues.

## Pull Request Signals

The PR or MR must show:

- a `DiffPal Review Summary` review, discussion, or thread;
- inline comments, discussions, or PR threads when actionable findings exist;
- a check, commit status, or PR status for the DiffPal review job.

## Required Artifacts

The CI workspace must contain:

| Path | Purpose |
| --- | --- |
| `.artifacts/diffpal/findings.json` | Canonical structured findings bundle. |
| `.artifacts/diffpal/summary.md` | Human-readable review summary. |

Depending on the host and feedback mode, the run may also write:

| Path | Purpose |
| --- | --- |
| `.artifacts/diffpal/diffpal.sarif` | SARIF report when enabled by the platform output. |
| `.artifacts/diffpal/codequality.json` | GitLab Code Quality report. |

## No Findings Vs Broken Review

`No findings` is healthy when:

- the review summary appears;
- `findings.json` exists and is valid JSON;
- the CI job succeeds unless a configured gate blocks it;
- logs show DiffPal resolved the review scope and completed publishing.

A broken review usually has one of these symptoms:

- no summary appears in the PR/MR;
- `findings.json` is missing;
- provider authentication failed;
- the workflow used a shallow checkout and could not compare base and head;
- the platform token could not publish feedback.

## Gate Behavior

When gating is enabled, DiffPal fails the CI job if at least one finding meets
`diffpal.gate.block_on`. Setup, authentication, review scope resolution, and
publishing failures also fail the job because the review result is incomplete.

When gating is disabled, blocking findings can still be published as review
feedback, but they do not fail the CI job. Tooling failures still fail because
DiffPal did not produce a complete review.

## Troubleshooting

- Missing summary or inline comments:
  [Review Completes But No Summary Appears](../help/troubleshooting.md#review-completes-but-no-summary-appears)
- Missing or incomplete diff:
  [DiffPal Cannot Find The Base Or Head Revision](../help/troubleshooting.md#diffpal-cannot-find-the-base-or-head-revision)
- Provider auth failures:
  [Provider Authentication Fails](../help/troubleshooting.md#provider-authentication-fails)
- Gate failures:
  [Gate Blocks Unexpectedly](../help/troubleshooting.md#gate-blocks-unexpectedly)
- Fork PR secrets:
  [Secrets and fork PRs](../guides/secrets-and-fork-prs.md)

After the first run is healthy, continue with [Next Steps](next-steps.md).
