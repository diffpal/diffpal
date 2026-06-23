# What Success Looks Like

After the first successful DiffPal run, the pull request should show review
feedback in the host UI and the CI job should write machine-readable artifacts.

## Pull Request Feedback

- A `DiffPal Review Summary` or equivalent summary thread for the PR/MR.
- Inline comments, discussions, or PR threads when actionable findings exist on
  changed lines.
- A check, commit status, or PR status named for the DiffPal review job.

## Artifacts

| Path | Purpose |
| --- | --- |
| `.artifacts/diffpal/findings.json` | Canonical structured findings bundle. |
| `.artifacts/diffpal/summary.md` | Human-readable review summary. |
| `.artifacts/diffpal/diffpal.sarif` | SARIF report when enabled by the host output. |
| `.artifacts/diffpal/codequality.json` | GitLab Code Quality report. |

The findings bundle includes prompt metadata such as prompt id, prompt version,
purpose, and findings schema version. That makes review output auditable even
when teams use different providers.

## Gate Behavior

DiffPal fails the CI job only when `--gate` or the matching action/task input is
enabled and at least one finding meets `diffpal.gate.block_on`. Setup,
authentication, diff collection, and publishing failures also fail the job
because they mean the review result is incomplete.

Start with `block_on: high` for early rollout. Lower the threshold only after
your team has tuned review instructions and provider behavior.

## If You Do Not See This

Use [troubleshooting](troubleshooting.md) to check platform permissions, fork PR
secret exposure, full git history, base/head resolution, and provider
authentication.
