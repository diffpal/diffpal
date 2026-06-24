# Custom CI/CD

Use this page to run DiffPal from Jenkins, Buildkite, CircleCI, Bitbucket
Pipelines, an internal runner, or any other CI system. This is a portable
integration contract, not native product support for those CI systems.

## Integration Modes

| Mode | What DiffPal does |
| --- | --- |
| Artifact-only review | Runs review, writes artifacts, and prints Markdown output without publishing to a code host. |
| Review plus publishing | Runs review and publishes feedback to GitHub, GitLab, or Azure DevOps when the job has host credentials. |
| Review with a blocking gate | Runs review and returns a non-zero result when blocking findings meet the configured threshold. |

Native publishers exist for GitHub, GitLab, and Azure DevOps. Other code hosts
can still receive artifact-only output from `diffpal review local`, but DiffPal
does not publish native comments or statuses to unsupported hosts.

## Required Inputs

Your CI job must provide:

- repository working tree;
- full git history for the reviewed range;
- base and head revisions;
- DiffPal configuration at `.config/diffpal/config.yaml`;
- provider executable and credentials;
- host credentials when publishing to GitHub, GitLab, or Azure DevOps.

## Generic Execution Lifecycle

1. Checkout the repository.
2. Fetch enough git history to resolve base and head.
3. Install the provider executable.
4. Authenticate the provider.
5. Install DiffPal.
6. Validate required environment variables and revisions.
7. Execute review.
8. Publish output when using a native host publisher.
9. Upload `.artifacts/diffpal/`.
10. Enforce the process exit result.

## Base And Head Resolution

DiffPal needs two revisions:

| Value | Meaning |
| --- | --- |
| `base` | The target branch revision or merge-base used as the comparison start. |
| `head` | The pull request, merge request, or candidate commit being reviewed. |

Your CI system must supply these values from pull-request metadata or from its
own build variables. Shallow clones, missing target branches, and deleted base
refs break diff collection because DiffPal cannot compute the reviewed change
range. Fetch full history or explicitly fetch the target branch before running
DiffPal.

## Portable Example

This vendor-neutral shell example performs an artifact-only review. It uses the
current CLI command and flags for `diffpal review local`. The provider setup
shown is the Codex API-key path; use [Providers](../providers/README.md) to
replace those install and authentication commands for another provider.

```bash
set -eu

: "${DIFFPAL_BASE_REV:?set DIFFPAL_BASE_REV to the target/base revision}"
: "${DIFFPAL_HEAD_REV:?set DIFFPAL_HEAD_REV to the reviewed/head revision}"
: "${DIFFPAL_REPO_ID:?set DIFFPAL_REPO_ID to a stable repository id}"
: "${OPENAI_API_KEY:?set OPENAI_API_KEY for the Codex provider}"
```

`OPENAI_API_KEY` is a provider credential. Do not expose it to untrusted fork
builds or jobs that execute fork-controlled code.

```bash
git fetch --tags --prune
git rev-parse --verify "$DIFFPAL_BASE_REV^{commit}"
git rev-parse --verify "$DIFFPAL_HEAD_REV^{commit}"

npm install --global @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3
printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
npm install --global @diffpal/diffpal@latest
mkdir -p .artifacts/diffpal

diffpal --profile ci review local \
  --base "$DIFFPAL_BASE_REV" \
  --head "$DIFFPAL_HEAD_REV" \
  --repo "$DIFFPAL_REPO_ID" \
  --review-id "custom-ci-${DIFFPAL_HEAD_REV}" \
  --feedback review \
  --out .artifacts/diffpal/findings.json \
  > .artifacts/diffpal/summary.md
```

To publish to a native host, use the matching command and host credentials:

- `diffpal review github`
- `diffpal review gitlab`
- `diffpal review ado`

Pass `--gate` when blocking findings should fail the CI job.

## CI Concept Mapping

| CI concept | Capability to configure |
| --- | --- |
| Checkout | Clone the repository and fetch the target/base revision plus the reviewed/head revision. |
| Protected secrets | Expose provider credentials, and host credentials when publishing, only to trusted jobs. |
| Pull-request metadata | Provide stable base revision, head revision, repository id, and review id. |
| Artifact upload | Persist the files created by the selected review command. |
| Required check/gate | Use the DiffPal process exit result as the CI job result when `--gate` is enabled. |

## Outputs

Verified local artifact paths for the portable `review local` example:

| Path | Purpose |
| --- | --- |
| `.artifacts/diffpal/findings.json` | Canonical structured findings bundle. |
| `.artifacts/diffpal/summary.md` | Human-readable review summary captured from stdout. |

Native host publishers can create additional platform artifacts:

| Path | Purpose |
| --- | --- |
| `.artifacts/diffpal/diffpal.sarif` | SARIF report when enabled by the platform output. |
| `.artifacts/diffpal/codequality.json` | GitLab Code Quality report. |

Publishing is available only through the native GitHub, GitLab, and Azure
commands. `feedback` controls what DiffPal tries to publish or print:
`summary` omits file-level findings, while `review` includes file-level
feedback where the selected mode supports it.

Gating is separate from feedback. `--gate` controls whether blocking findings
make the process return non-zero. Feedback can still show blocking findings
when the gate is disabled.

## Expected Results

The portable artifact-only example should create
`.artifacts/diffpal/findings.json` and a captured
`.artifacts/diffpal/summary.md`. Native host commands should additionally
publish the feedback surfaces listed in the
[support matrix](../reference/support-matrix.md).

## Security

Do not expose provider or host secrets to untrusted fork builds. Run
secret-backed review only for trusted branches, same-repository pull requests,
or maintainer-approved jobs that do not execute untrusted code with secrets.

For untrusted contributions, run no-secret CI checks only, or require a
maintainer-controlled rerun after reviewing the submitted workflow changes.
Artifact-only review still runs a provider-backed review, so it must not receive
provider credentials in a job that executes untrusted fork code.

Use [Secrets and fork PRs](../guides/secrets-and-fork-prs.md) for the full
credential and fork PR guide.

## Failure Handling

DiffPal returns success when the review completes and no enabled gate blocks the
job. With `--gate`, blocking findings return a non-zero process result.
Configuration, provider authentication, diff collection, and publishing
failures also return non-zero because the review result is incomplete.

Treat the process exit result as the stable public contract. Do not depend on
specific numeric exit codes unless they are documented as stable for the
command you are using.

## Boundaries

DiffPal has native publishers for:

- GitHub;
- GitLab;
- Azure DevOps.

When running against another code host, DiffPal can still produce local
Markdown and artifacts, including `findings.json`, but it does not create
native comments, discussions, threads, checks, statuses, or branch-policy
signals for that host.

Next step: upload `.artifacts/diffpal/` from your CI job and verify the first
run with [Verify First Review](../getting-started/verify-first-review.md).
