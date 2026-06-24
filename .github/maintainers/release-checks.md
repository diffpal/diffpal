# Maintainer Release Checks

This page keeps non-sensitive maintainer checks out of the public user docs.

## Versioning Model

- CLI and Go module releases use SemVer tags such as `v1.2.3`.
- The CLI is distributed through omnidist, including the npm package
  `@diffpal/diffpal`.
- Consumers should install the CLI with a pinned SemVer version when they need
  reproducible CI.

## Branch Policy Expectations

- `main` remains the release source branch.
- Major releases should keep backward compatible CLI flags where possible.
- Breaking API changes require a major version bump and a migration note in
  user docs.

## CI Baseline

- The `ci` workflow provides lint, test, and security jobs.
- Lint checks module integrity, formatting, vetting, static analysis, workflow
  syntax, and the CLI help surface.
- Test coverage includes `go test ./...`; race testing should run before
  release promotion.

## Documentation Sync Points

Before a release, confirm these public docs still match the CLI surface:

- `docs/getting-started/github-quickstart.md`
- `docs/reference/configuration.md`
- `docs/integrations/README.md`
- `docs/integrations/gitlab-ci.md`
- `docs/integrations/azure-pipelines.md`
