# Custom CI

Use this contract for Jenkins, Buildkite, CircleCI, Bitbucket Pipelines, internal runners, and other CI systems. DiffPal provides native publishing only for GitHub, GitLab, and Azure DevOps; other hosts use artifact-only local review.

## Required lifecycle

1. Check out the repository and fetch full history or explicitly fetch both reviewed revisions.
2. Reject or defer any secret-backed job that executes untrusted fork-controlled code.
3. Install a deliberately selected DiffPal version and the configured provider version.
4. Authenticate the provider from the CI secret store without printing its value.
5. Validate DIFFPAL_BASE_REV, DIFFPAL_HEAD_REV, and DIFFPAL_REPO_ID; derive a stable review ID.
6. Run provider-free validation, then the selected review command.
7. Upload .artifacts/diffpal/ even when the review job fails.
8. Preserve the DiffPal process status so an enabled gate can fail the job.

## Artifact-only command

Use diffpal --profile ci review local with:

- base from DIFFPAL_BASE_REV;
- head from DIFFPAL_HEAD_REV;
- repo from DIFFPAL_REPO_ID;
- a stable CI review ID;
- explicit --feedback;
- output .artifacts/diffpal/findings.json;
- stdout captured as .artifacts/diffpal/summary.md;
- optional --gate.

Artifact-only mode still sends review context to the selected provider. Protect its credential, for example OPENAI_API_KEY, from untrusted contributions.

## Native-host publication

For a GitHub, GitLab, or Azure DevOps repository, publication may instead use review github, review gitlab, or review ado. Add the matching host metadata and host credential (GITHUB_TOKEN, CI_JOB_TOKEN or GITLAB_TOKEN, or SYSTEM_ACCESSTOKEN) separately from the provider credential. Explain the feedback surfaces before enabling publication.

## Merge and preview checks

Adapt the lifecycle to the CI system's native checkout, secret, artifact, and required-check primitives. Preserve existing stages and failure handling. Preview selected versions, variable names, base/head mapping, trust policy, feedback, gate, and artifact retention.

Do not create secrets, push, trigger a build, or publish a review. Report exact administrator steps for the chosen CI product after validating its local configuration syntax where tooling exists.
