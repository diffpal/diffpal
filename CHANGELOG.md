# Changelog

## Unreleased

### Breaking changes

- Removed `diffpal init --state` and generation of
  `.config/diffpal/state/`. Host review state is reconciled directly from the
  active pull request or merge request.

### Added

- Added deterministic live-thread reconciliation for GitHub, GitLab, and Azure
  DevOps, including stale finding resolution.
- Added bounded platform HTTP clients, structured platform errors, atomic
  private artifact writes, and cancellation-aware Git collection.
- Added robust diff parsing for long lines, quoted and Unicode paths, renames,
  deletions, and binary changes.
- Added a 75% coverage gate and stricter context, HTTP, SQL, error, revive, and
  cognitive-complexity lint checks.
- Added a v1 migration guide and explicit trusted-code guidance for autonomous
  credentialed providers.

### Changed

- Changed credentialed CI examples to pin DiffPal `1.0.0`; interactive
  onboarding may continue using the `latest` tag.
- Simplified the runtime by removing unused SQLite cache, idempotency, and
  telemetry implementations.

### Fixed

- Fixed empty in-memory finding IDs that caused host markers to lose stable
  identity.
- Fixed GitLab and Azure artifact state loaders to understand their documented
  output envelopes.
- Fixed reruns that could duplicate Azure threads or GitHub and GitLab findings.
