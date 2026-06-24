# Versioning

DiffPal uses SemVer for user-facing releases.

## CLI And Package Versions

The CLI version is built into release binaries with Omnidist ldflags:

- `Version`
- `GitCommit`
- `BuildDate`

`diffpal version` prints:

```text
diffpal <version>+<git-commit> (<build-date>)
```

The npm package `@diffpal/diffpal` is the public CLI package. Omnidist also
builds platform-specific packages and release binaries from the same version.
Consumers should pin exact versions in credentialed CI jobs when repeatability
matters.

GitHub Action users can use the stable major tag, for example
`diffpal/action@v1`, or pin a more specific release. Azure users can use the
published DiffPal extension task version.

## Configuration Schema Compatibility

Configuration files use `version: v1`. DiffPal accepts an empty version or
`v1`; any other config version is rejected.

Minor releases should preserve `v1` compatibility. New optional fields may be
added without requiring existing configs to change.

## Artifact And Schema Compatibility

New findings writes use bundle `version: v3`. DiffPal readers accept `v1`,
`v2`, and `v3` findings bundles.

Consumers should:

- treat `findings.json` as the canonical artifact;
- ignore unknown fields;
- use `version` and `prompt.schema_version` when they need compatibility
  branching;
- avoid depending on host-specific publishing plan files unless they are part of
  the selected integration.

## Upgrade Guidance

1. Pin the DiffPal CLI, provider CLI, and CI wrapper versions in credentialed
   jobs.
2. Upgrade in a branch and run `diffpal doctor --mode <host>`.
3. Run a same-repository PR/MR review before enabling gates on the new version.
4. Keep uploaded artifacts for the first upgraded runs so schema or publisher
   differences are easy to inspect.
