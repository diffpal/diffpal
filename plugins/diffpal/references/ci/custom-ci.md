# Custom CI

Use this contract for Jenkins, Buildkite, CircleCI, Bitbucket Pipelines, internal runners, and other CI systems. DiffPal has native publishing commands for GitHub, GitLab, and Azure DevOps; other hosts use artifact-only review.

## Required lifecycle

1. Check out full history or fetch both reviewed revisions explicitly.
2. Validate `DIFFPAL_BASE_REV`, `DIFFPAL_HEAD_REV`, and `DIFFPAL_REPO_ID`; derive a stable review ID.
3. Select deliberate DiffPal and provider versions using repository policy.
4. Establish the trust boundary before any provider credential or contributor-controlled command is available.
5. Run `<diffpal-command> --profile ci doctor --mode local` with the same executable and config source as review.
6. Authenticate the provider from the CI secret store without printing its value.
7. Run the selected review command, capture Markdown stdout, and retain `.artifacts/diffpal/` even on failure.
8. Preserve the DiffPal process status so an enabled gate can fail the job.

## Trust and configuration

Fork and same-repository tests are filters, not proof that the checked-out source is trusted. Source changes can replace `.config/diffpal/config.yaml` and its provider `cmd`. Require a CI-native maintainer approval/trusted-author policy, or materialize config from a trusted base ref or secured external source.

For external config, place `diffpal/config.yaml` or `config.yaml` under a trusted root, verify that exact file exists, and pass `--config-dir <trusted-config-root>` to doctor and review. The existence check prevents fallback to workspace config. Do not execute source-controlled hooks, package scripts, or provider commands before establishing trust.

## Artifact-only mode

Use `review local`; do not provide a host publication token:

```bash
<diffpal-command> --profile ci review local \
  --base "$DIFFPAL_BASE_REV" \
  --head "$DIFFPAL_HEAD_REV" \
  --repo "$DIFFPAL_REPO_ID" \
  --review-id "$DIFFPAL_REVIEW_ID" \
  --feedback <summary-or-review> \
  --out .artifacts/diffpal/findings.json
```

Append `--gate` when gating is enabled.

This still sends review context to the configured provider, so its credential belongs behind the trust guard. Retain findings and captured Markdown stdout as CI artifacts.

## Native publishing mode

For GitHub, GitLab, or Azure DevOps, use `review github`, `review gitlab`, or `review ado` with explicit base, head, repository, review ID, feedback, output, and gate. Pass the matching host credential separately from the provider credential and only to the publication step. Explain the host outputs before enabling them.

For any other host, keep artifact-only mode unless a separately reviewed publisher consumes the findings bundle.

## Merge and handoff

Adapt checkout, approvals, secrets, artifacts, and required-check behavior to the CI product while preserving existing stages and failure handling. Preview versions, variable names, revision mapping, trust predicate, config origin, feedback, gate, host permissions, and retention. Do not create secrets, push, trigger a build, approve a gate, or publish during setup.
