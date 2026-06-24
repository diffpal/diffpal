# Versioning

DiffPal uses SemVer for the CLI and Go module. npm consumers can pin
`@diffpal/diffpal` by SemVer or use the `latest` dist-tag.

GitHub Action consumers track the stable major tag, for example
`diffpal/action@v1`, or pin a more specific release when required by their
release policy.

Configuration files use `version: v1` at the top level. Existing findings
bundles may use older bundle versions; new review runs write the current
findings schema described in the [findings schema](findings-schema.md).

DiffPal is distributed as a CLI. Packages under `internal/*` are private
implementation details and are not a supported public Go API.
