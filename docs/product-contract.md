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

- Public promise: open-source AI PR review you control.
- Hero message: bring your own agent; keep one PR review workflow.
- Product category: AI pull request review control plane, with GitHub, GitLab,
  and Azure DevOps as supported publishing targets.
- Provider model: users bring their preferred AI provider or ACP-compatible CLI
  through `runtime.providers`; DiffPal does not require a hosted DiffPal review
  service.
- Affordability story: teams keep cost control with their existing provider
  choices instead of adopting a required per-seat review platform.
- DiffPal product surface: diff collection, structured findings, summaries,
  file-level feedback, artifacts, and merge gates.
- Differentiation: provider choice, repository-owned policy, machine-readable
  artifacts, and cross-host publishing consistency.

## Project Goals

- Make AI PR review open, portable, controllable, and CI-native.
- Let teams choose the provider path and cost model that fits them.
- Provide consistent review outputs and merge gates across GitHub, GitLab, and
  Azure DevOps.
- Avoid a mandatory hosted DiffPal service or required per-seat review
  platform.

## Host support matrix

| Host | Phase | Primary surfaces |
|---|---|---|
| GitHub | MVP | PR reviews, file-level review comments, markdown summary, SARIF |
| GitLab | v1 | discussions, commit status, Code Quality, SARIF |
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
- GitHub review comment plan: `.artifacts/diffpal/github-comments.json`
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
- Language of CLI defaults to `review` flows and findings JSON outputs.
- Primary review subcommands are `local`, `github`, `gitlab`, and `ado`.
- User-facing host output behavior is configurable by review `--feedback`.
- Merge gating is based on workflow exit status or platform status surfaces, not bot approval semantics.
