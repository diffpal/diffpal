# Troubleshooting

Use this page when a DiffPal run does not produce the expected host feedback,
artifacts, or gate result. Each entry starts from what you can observe in CI or
on the pull request.

## Job Does Not Start

**Symptom:** The DiffPal workflow, job, or task is not created for the pull
request or merge request.

**Likely causes:** The CI trigger does not include PR/MR events, the PR is a
draft, a same-repository or trusted-review guard skipped the job, the workflow
file is not on the default branch, or the CI system is disabled for the
repository.

**Diagnostic:** Check the CI event and guard expression first. For GitHub
Actions, confirm the workflow has a `pull_request` trigger and inspect whether
the job condition matches the PR:

```yaml
if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
```

**Fix:** Enable the PR/MR trigger for the selected CI system. Keep fork-safety
guards in place, but test the first run with a same-repository, non-draft pull
request.

**Related:** [GitHub quickstart](../getting-started/github-quickstart.md),
[GitHub Actions](../integrations/github-actions.md), and
[Secrets and fork PRs](../guides/secrets-and-fork-prs.md).

## Provider Authentication Fails

**Symptom:** The job starts, but provider login or provider validation fails
before review output is produced.

**Likely causes:** The provider CLI is not installed, the selected
`diffpal.provider` entry does not match an installed/authenticated provider,
the provider secret is missing in the trusted job, or the secret belongs to a
different provider account.

**Diagnostic:** Run the host-specific doctor after provider installation and
authentication:

```bash
diffpal --profile ci doctor --mode github
```

Use `--mode gitlab`, `--mode ado`, or `--mode local` for other runs.

**Fix:** Install the provider command before DiffPal runs, expose the provider
secret only to trusted jobs, and follow the selected provider page for the
exact authentication command.

**Related:** [Providers](../providers/README.md),
[CLI doctor](../reference/cli.md#diffpal-doctor), and
[Secrets and fork PRs](../guides/secrets-and-fork-prs.md).

## DiffPal Cannot Find The Base Or Head Revision

**Symptom:** The review fails during diff collection with missing, ambiguous,
or invalid base/head revisions.

**Likely causes:** The checkout is shallow, the target branch was not fetched,
CI metadata is unavailable, `--base` or `--head` is empty, or the job is not
running in a PR/MR context.

**Diagnostic:** Verify the revisions in the CI workspace:

```bash
git rev-parse --verify "$BASE_SHA^{commit}"
git rev-parse --verify "$HEAD_SHA^{commit}"
git merge-base "$BASE_SHA" "$HEAD_SHA"
```

**Fix:** Use full checkout history. In GitHub Actions set `fetch-depth: 0`; in
GitLab CI set `GIT_DEPTH: "0"`; in Azure Pipelines set `fetchDepth: 0`. For
custom CI, explicitly fetch the target branch and pass stable `--base` and
`--head` values.

**Related:** [GitHub Actions checkout](../integrations/github-actions.md#required-checkout-behavior),
[GitLab checkout](../integrations/gitlab-ci.md#required-checkout-behavior),
[Azure checkout](../integrations/azure-pipelines.md#required-checkout-behavior),
and [Custom CI base/head resolution](../integrations/custom-ci.md#base-and-head-resolution).

## Review Completes But No Summary Appears

**Symptom:** The provider-backed review finishes, but the PR/MR does not show a
DiffPal summary.

**Likely causes:** Host publishing was denied, the wrong host command was used,
`diffpal review local` printed Markdown to stdout without publishing, GitHub
`--dry-run` was used, or `summary.md` was not retained from artifacts.

**Diagnostic:** Check whether the command wrote a summary artifact:

```bash
test -s .artifacts/diffpal/summary.md
```

For local review, check the redirected stdout file if your CI captures one.

**Fix:** Use the matching host command or action for GitHub, GitLab, or Azure
DevOps. Ensure the host token is present with write permission and upload
`.artifacts/diffpal/` for retained summaries.

**Related:** [Artifacts reference](../reference/artifacts.md),
[Support matrix](../reference/support-matrix.md), and
[Integrations](../integrations/README.md).

## Summary Appears But Inline Findings Do Not

**Symptom:** The PR/MR has a DiffPal summary, but no file-level comments,
discussions, or threads appear.

**Likely causes:** `feedback` is set to `summary`, the host does not support
inline feedback for the selected mode, findings could not be anchored to
changed lines, or the provider returned no actionable file-level findings.

**Diagnostic:** Confirm the run used review feedback:

```bash
diffpal --profile ci review local --base origin/main --head HEAD --feedback review
```

Inspect `.artifacts/diffpal/findings.json` to see whether findings have changed
file locations.

**Fix:** Set `feedback: review` or pass `--feedback review`. If findings exist
without changed-line locations, review the provider output and diff range; only
changed-file locations can become inline host feedback.

**Related:** [Feedback modes](../integrations/README.md#feedback-modes),
[Findings, feedback, and gates](../concepts/findings-feedback-and-gates.md),
and [Findings schema](../reference/findings-schema.md).

## No Findings Were Produced

**Symptom:** The review summary says no actionable findings were found, or the
findings bundle contains an empty findings list.

**Likely causes:** The change has no issues that meet the review contract, the
diff range is too small or wrong, provider instructions are too narrow, or the
provider returned a valid no-findings result.

**Diagnostic:** Confirm that the reviewed diff is the expected range:

```bash
git diff --stat "$BASE_SHA" "$HEAD_SHA"
```

Then inspect `.artifacts/diffpal/findings.json`.

**Fix:** If the diff range is wrong, fix base/head resolution. If the diff is
right, treat no findings as a completed review and tune repository instructions
only when the provider is consistently missing issues your team expects.

**Related:** [Verify First Review](../getting-started/verify-first-review.md),
[Review lifecycle](../concepts/review-lifecycle.md), and
[Configuration review settings](../reference/configuration.md#review-settings).

## Publishing Is Denied

**Symptom:** Review generation succeeds, but publishing comments, discussions,
threads, statuses, SARIF, or reports fails.

**Likely causes:** The host token is missing, read-only, scoped to the wrong
repository/project, unavailable to the job, or the host API URL is wrong for an
enterprise/self-managed instance.

**Diagnostic:** Run doctor in the host mode:

```bash
diffpal --profile ci doctor --mode gitlab
```

Use `github` or `ado` for the other native publishers.

**Fix:** Provide the host credential documented for the selected platform and
grant the minimum write permission needed for the selected feedback surface.
Keep host credentials separate from provider credentials.

**Related:** [Secrets and fork PRs](../guides/secrets-and-fork-prs.md),
[GitHub token permissions](../integrations/github-actions.md#required-token-and-minimum-permissions),
[GitLab tokens](../integrations/gitlab-ci.md#required-token-and-minimum-permissions),
and [Azure token setup](../integrations/azure-pipelines.md#required-token-and-minimum-permissions).

## Fork PRs Are Skipped

**Symptom:** DiffPal does not run, or only no-secret CI runs, for a fork pull
request.

**Likely causes:** The workflow intentionally protects provider and host
credentials from untrusted fork code. GitHub, GitLab, or Azure guards can skip
credentialed review for fork contributions.

**Diagnostic:** Inspect the CI condition or rules. GitHub examples use:

```yaml
if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
```

Azure examples use `System.PullRequest.IsFork`; GitLab examples use
same-project MR conditions and a maintainer-controlled trusted-review guard.

**Fix:** Keep the skip for untrusted fork code. For external contributions, run
no-secret CI or use a maintainer-approved workflow that does not execute
fork-controlled code with secrets.

**Related:** [Secrets and fork PRs](../guides/secrets-and-fork-prs.md),
[GitHub Actions fork behavior](../integrations/github-actions.md#fork-or-untrusted-contribution-behavior),
[GitLab fork behavior](../integrations/gitlab-ci.md#fork-or-untrusted-contribution-behavior),
and [Azure fork behavior](../integrations/azure-pipelines.md#fork-or-untrusted-contribution-behavior).

## Expected Artifacts Are Missing

**Symptom:** The CI job finishes, but `.artifacts/diffpal/findings.json`,
`summary.md`, or host-specific artifacts are not available after the job.

**Likely causes:** The command failed before writing artifacts, `--out` changed
the findings path, `diffpal review local` printed Markdown to stdout instead
of creating `summary.md`, CI did not upload `.artifacts/diffpal/`, or a dry-run
did not create host artifacts.

**Diagnostic:** List the artifact directory inside the job:

```bash
find .artifacts/diffpal -maxdepth 1 -type f -print
```

**Fix:** Upload `.artifacts/diffpal/` from CI. For local review, redirect
stdout to `.artifacts/diffpal/summary.md` when you want a retained summary.
Check `--out` if `findings.json` was intentionally moved.

**Related:** [Artifacts reference](../reference/artifacts.md) and
[Custom CI outputs](../integrations/custom-ci.md#outputs).

## SARIF Or Code Quality Output Is Missing

**Symptom:** The review completes, but `.artifacts/diffpal/diffpal.sarif` or
`.artifacts/diffpal/codequality.json` is absent.

**Likely causes:** SARIF is not a native Azure surface, Code Quality is a
GitLab-specific output, the run used `review local`, GitHub dry-run was used,
or the file was not uploaded as a CI artifact/report.

**Diagnostic:** Check the selected host and command against the artifact table.
To create SARIF from a findings bundle manually, run:

```bash
diffpal sarif --input .artifacts/diffpal/findings.json --out .artifacts/diffpal/diffpal.sarif
```

**Fix:** Use GitHub or GitLab host review when you expect SARIF from a native
run. Use GitLab CI report configuration for Code Quality. For custom CI or
Azure, convert findings to SARIF explicitly when needed.

**Related:** [Artifacts reference](../reference/artifacts.md),
[Support matrix](../reference/support-matrix.md), and
[CLI SARIF command](../reference/cli.md#diffpal-sarif).

## Gate Blocks Unexpectedly

**Symptom:** The job fails with blocking findings even though publishing and
artifacts completed.

**Likely causes:** `--gate` or `gate: true` is enabled, and at least one
finding severity meets `diffpal.gate.block_on` or `--block-on`.

**Diagnostic:** Check the process exit code and configured threshold. Blocking
findings with an enabled gate return exit code `10`.

**Fix:** Inspect `findings.json` and the review summary. If the finding should
not block, raise `diffpal.gate.block_on` to a higher severity or improve review
instructions. If it should block, fix the code.

**Related:** [Exit behavior](../reference/exit-behavior.md),
[Configuration gate](../reference/configuration.md#gate), and
[Findings, feedback, and gates](../concepts/findings-feedback-and-gates.md).

## Gate Does Not Block

**Symptom:** Findings appear in the summary or comments, but the CI job still
passes.

**Likely causes:** The gate is disabled, findings are below the blocking
threshold, the command did not pass `--gate`, or the CI wrapper is not using the
DiffPal process result as the job result.

**Diagnostic:** Confirm the run enables the gate and check the threshold:

```bash
diffpal --profile ci review local --base origin/main --head HEAD --gate
```

**Fix:** Enable `gate: true` in the action/task or pass `--gate` to the CLI.
Set `diffpal.gate.block_on` to the minimum severity that should fail the job.

**Related:** [Merge gates](../integrations/README.md#merge-gates),
[Exit behavior](../reference/exit-behavior.md), and
[Configuration gate](../reference/configuration.md#gate).

## Local Execution Works But CI Execution Fails

**Symptom:** `diffpal review local` works on a developer machine, but the same
repository fails in CI.

**Likely causes:** CI has a shallow checkout, missing provider installation,
missing or protected secrets, different profile selection, missing host
metadata, a different working directory, or no artifact upload step.

**Diagnostic:** Compare local and CI environments with doctor and version:

```bash
diffpal version
diffpal --profile ci doctor --mode local
```

In host CI, use the matching host mode.

**Fix:** Pin the same provider install path in CI, use full checkout history,
make the intended profile explicit, expose secrets only to trusted jobs, and
pass host metadata or base/head values from CI variables.

**Related:** [CLI reference](../reference/cli.md),
[Custom CI lifecycle](../integrations/custom-ci.md#generic-execution-lifecycle),
and [Providers](../providers/README.md).

Next step: after applying the fix, rerun `diffpal doctor` in the same CI mode
that failed.
