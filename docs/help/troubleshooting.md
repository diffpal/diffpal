# Troubleshooting

Use this page when DiffPal ran but the pull request did not show the expected
summary, comments, status, or artifacts.

## Fork Pull Requests And Secrets

Do not expose provider credentials to untrusted fork PR code. Fork workflow
approval settings control whether external workflows run automatically; they do
not make it safe to release provider secrets to fork code.

Keep secret-backed DiffPal review limited to same-repository pull requests,
trusted branches, or maintainer-controlled automation that does not execute fork
code. Fork PRs should run no-secret CI only.

For GitHub Actions, keep a guard like:

```yaml
if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
```

Do not combine `pull_request_target` with checking out the PR head or running
package installs, tests, build scripts, hooks, provider CLIs, or other fork
code.

## Missing Summary Or Inline Comments

- GitHub: confirm `pull-requests: write` and that `GITHUB_TOKEN` is available on
  the review step.
- GitLab: use `GITLAB_TOKEN` when `CI_JOB_TOKEN` does not have enough API
  access for merge request publishing.
- Azure DevOps: enable **Allow scripts to access the OAuth token** and pass
  `SYSTEM_ACCESSTOKEN` to the `DiffPalReview@1` task.
- Confirm `feedback` is set to `review` when you expect file-level comments.
  `summary` intentionally publishes only the summary and non-file artifacts.

## Missing Or Incomplete Diff

DiffPal needs enough git history to compare base and head commits.

- GitHub Actions: use `fetch-depth: 0`.
- GitLab CI: use `GIT_DEPTH: "0"`.
- Azure Pipelines: use `fetchDepth: 0`.

Run from a pull request or merge request pipeline when possible. For advanced
manual runs, pass explicit `base` and `head` revisions.

## Provider Authentication Fails

- Confirm the provider CLI is installed before the DiffPal step.
- Confirm the provider auth secret is present only in trusted pipelines.
- For Codex API-key auth, set `OPENAI_API_KEY` and run
  `codex login --with-api-key`.
- For Codex subscription auth, regenerate `CODEX_AUTH_JSON_B64` with the recipe
  in [examples](../../examples/README.md#generate-codexauthjsonb64).
- For Copilot, use a fine-grained GitHub PAT v2 with the Copilot Requests
  permission.

## Gate Fails The Job

When `--gate` is enabled, DiffPal fails the job if any finding meets
`diffpal.gate.block_on`. Start with `block_on: high` while tuning provider
instructions. Lower the threshold only after the team agrees that medium or low
findings should block merges.

Setup, authentication, diff collection, and publishing failures also fail the
job because the review result is incomplete.

## Local Debugging

Run the host-specific doctor before enabling a blocking gate:

```bash
diffpal doctor --mode github
```

Render a local Markdown report while still writing the findings bundle:

```bash
diffpal review local --base origin/main --head HEAD --profile ci
```

Use `--feedback summary` when you want the summary without file-level detailed
comments. Redirect stdout to save the Markdown report to a file.

Inspect the prompt contract without spending provider quota:

```bash
diffpal debug prompt --base origin/main --head HEAD --profile ci --format text
```

The debug command loads config, collects the git diff, runs review
normalization, and writes a schema-valid mock findings bundle without making a
provider call.
