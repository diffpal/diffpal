# GitHub Actions

Merge DiffPal into an existing workflow under .github/workflows/; create a focused workflow only when no suitable file exists.

## Required shape

- Trigger on pull_request, not pull_request_target, for code review.
- Restrict the credentialed job to non-draft, same-repository pull requests:

      if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}

- Use actions/checkout@v4 with fetch-depth: 0.
- Grant contents: read; add pull-requests: write only for native GitHub publishing. Retain stricter existing permissions when sufficient.
- Install and authenticate the provider before review. Follow repository pinning policy; for the checked-in Codex API-key recipe, current verified provider packages are @openai/codex@0.139.0 and @normahq/codex-acp-bridge@1.8.4.
- Use diffpal/action@v1 with:
  - profile: ci
  - base from github.event.pull_request.base.sha
  - head from github.event.pull_request.head.sha
  - repo from github.repository
  - review ID derived from github.event.pull_request.number
  - explicit feedback and gate
- Pass the provider secret, such as OPENAI_API_KEY, separately from the host GITHUB_TOKEN. Do not copy either value into workflow text.
- Upload .artifacts/diffpal/ with actions/upload-artifact@v4 under if: always() when the repository does not already retain DiffPal outputs.

## Merge and preview checks

Preserve existing events, concurrency, jobs, matrices, and permissions. If adding concurrency, scope it to the pull-request number and avoid changing unrelated groups. Show whether the selected feedback publishes a summary or file-level review comments and whether the gate can fail the required check.

Do not configure secrets, push the workflow, dispatch it, or approve publication. In the handoff, name the provider secret and explain that GitHub supplies GITHUB_TOKEN; recommend a no-secret job or maintainer-controlled rerun for forks.
