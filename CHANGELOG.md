# Changelog

## Unreleased

- Added the DiffPal CLI command set and findings bundle runtime.
- Added GitHub, GitLab, and Azure adapter contracts and platform-specific task wrappers.
- Added the composite GitHub Action wrapper for `diffpal review github`.
- Added GitHub review channels so parallel DiffPal workflows publish separate checks and summary comments.
- Added the Azure DevOps `DiffPalReview@1` and `DiffPalReviewDev@1` extension packaging.
- Added the `omnidist-release` GitHub workflow for npm-only `@diffpal/diffpal` releases.
- Added the `diffpal-dev` GitHub workflow so DiffPal reviews same-repository pull requests using the locally built CLI, and left the released-package `diffpal` workflow disabled until the action is ready for PR gating.
- Added lint, test, race-test, actionlint, and govulncheck coverage for CI.
- Updated the GitHub Action default and CI examples to use the current `0.1.7` CLI release.
- Fixed Azure DevOps Marketplace packaging by adding the required extension overview asset.
- Fixed Azure DevOps Marketplace packaging by adding the required 128x128 extension icon asset.
- Fixed Azure DevOps Marketplace extension IDs to publish as `diffpal.ai-diff-review` and `diffpal.ai-diff-review-dev`.
- Fixed Azure DevOps Marketplace task UUIDs for the corrected extension identity.
