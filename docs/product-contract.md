# DiffPal Product Contract

## Foundation

- Product name: `DiffPal`
- Internal engine codename: `Aibolit`
- Default license: `MIT`
- Module path: `github.com/diffpal/diffpal`
- Primary support target: GitHub in MVP
- Secondary targets in v1: GitLab and Azure DevOps
- Hosted profile: local/CI-first

## Product Positioning

- Public promise: open-source, provider-agnostic AI review for pull requests.
- Product category: PR review, with GitHub, GitLab, and Azure DevOps as
  supported publishing targets.
- Provider model: users bring their own AI provider account or ACP-compatible
  CLI through `runtime.providers`; DiffPal does not require a hosted DiffPal
  review service.
- Affordability story: teams keep provider costs and credentials in accounts
  they own instead of adopting a required per-seat review platform.
- DiffPal-owned surface: diff collection, structured findings, summaries,
  inline feedback, artifacts, and merge gates.

## Host support matrix

| Host | Phase | Primary surfaces |
|---|---|---|
| GitHub | MVP | check runs, review comments, markdown summary, SARIF |
| GitLab | v1 | discussions, Code Quality, SARIF |
| Azure DevOps | v1 | PR threads, PR status |

## Binary and package surface

- CLI binary: `diffpal`
- Module package: `github.com/diffpal/diffpal`
- Action package: `diffpal/action`
- CLI npm package: `@diffpal/diffpal`
- NPM scope: `@diffpal/*`

## Artifact naming

- Diff findings bundle: `.artifacts/diffpal/findings.json`
- Markdown summary: `.artifacts/diffpal/summary.md`
- SARIF export: `.artifacts/diffpal/diffpal.sarif`
- Code Quality export: `.artifacts/diffpal/codequality.json`
- GitHub check-run payload: `.artifacts/diffpal/github-checkrun.json`
- GitHub inline comment plan: `.artifacts/diffpal/github-comments.json`
- GitLab discussions plan: `.artifacts/diffpal/gitlab-discussions.json`
- Azure threads plan: `.artifacts/diffpal/azure-threads.json`
- Azure status payload: `.artifacts/diffpal/azure-status.json`

## Versioning

- CLI and Go module are SemVer (`1.2.3`).
- Action major tag alias: `v1` (minor and patch tags are optional).
- npm package versions follow CLI SemVer.
- Configuration schema version includes `version: v1` at top-level.

## Runtime contract

- Go toolchain minimum: `1.26`
- Language of CLI defaults to `review` mode flows and findings JSON outputs.
- Primary review modes are `local`, `github`, `gitlab`, and `ado`.
- User-facing host output behavior is configurable by review `--feedback`;
  advanced publish surfaces remain configurable by `--mode`.
- Merge gating is based on `check/status` style surfaces, not bot approval semantics.
