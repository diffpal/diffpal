# Changelog

## Unreleased

- Added the DiffPal CLI command set and findings bundle runtime.
- Added GitHub, GitLab, and Azure adapter contracts and platform-specific task wrappers.
- Added the composite GitHub Action wrapper for `diffpal review github`.
- Added the Azure DevOps `DiffPalReview@1` and `DiffPalReviewDev@1` extension packaging.
- Added the `omnidist-release` GitHub workflow for npm-only `@diffpal/diffpal` releases.
- Added the `diffpal-review` GitHub workflow so DiffPal reviews same-repository pull requests using the npm-distributed CLI and this repository's action wrapper.
- Added lint, test, race-test, actionlint, and govulncheck coverage for CI.
