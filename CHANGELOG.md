# Changelog

## Unreleased

- Added the DiffPal CLI command set and findings bundle runtime.
- Added GitHub, GitLab, and Azure adapter contracts and platform-specific task wrappers.
- Added the composite GitHub Action wrapper for `diffpal review github`.
- Added GitHub review channels so parallel DiffPal workflows publish separate checks and PR reviews.
- Added the Azure DevOps `DiffPalReview@1` and `DiffPalReviewDev@1` extension packaging.
- Added the `omnidist-release` GitHub workflow for npm-only `@diffpal/diffpal` releases.
- Added the `diffpal-dev` GitHub workflow so DiffPal reviews same-repository pull requests using the locally built CLI, and left the released-package `diffpal` workflow disabled until the action is ready for PR gating.
- Added lint, test, race-test, actionlint, and govulncheck coverage for CI.
- Added a versioned prompt registry for Prompt Pack v1.2.0 so review prompt metadata, schema, and task rendering resolve from one registered prompt contract.
- Updated the GitHub Action default and CI examples to use the current `0.1.7` CLI release.
- Fixed Azure DevOps Marketplace packaging by adding the required extension overview asset.
- Fixed Azure DevOps Marketplace packaging by adding the required 128x128 extension icon asset.
- Fixed Azure DevOps Marketplace extension IDs to publish as `diffpal.diffpal` and `diffpal.diffpal-dev`.
- Fixed Azure DevOps Marketplace task UUIDs for the corrected extension identity.
- Fixed Azure DevOps Marketplace release cleanup for stale extension IDs.
- Fixed Azure DevOps Marketplace task UUIDs for the final `diffpal.diffpal` extension identity.
- Fixed Azure DevOps review threads to bind comments to the reviewed file and
  line range while removing duplicated severity/category metadata from thread
  bodies.
- Fixed Azure DevOps PR gate publishing to update the active reviewer vote with
  the single-reviewer API instead of resetting reviewer state through the bulk
  endpoint.
