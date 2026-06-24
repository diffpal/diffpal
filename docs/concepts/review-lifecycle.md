# Review Lifecycle

DiffPal review is a CI job lifecycle. Each stage has a distinct role so
failures are clear and artifacts are consistent.

## Diff Collection

DiffPal resolves the base and head revisions for the pull request, merge
request, or candidate commit. It collects the changed files, changed line
ranges, and related metadata that define the review scope.

Shallow clones, missing target branches, or incorrect base/head values usually
break this stage. Host-specific setup pages describe the required checkout
behavior.

## Review Request

DiffPal combines the diff metadata, repository configuration, review
instructions, language, and gate threshold into a provider request. The selected
provider or ACP-compatible agent inspects the requested `base..head` change with
its available tools.

## Structured Findings

The provider returns structured review output: a change summary and zero or
more findings. A finding describes a concrete issue, its severity, category,
impact, and the changed lines it applies to.

## Validation

DiffPal validates and normalizes provider output before publishing it. Findings
must map back to the reviewed change and the configured schema. Invalid or
unanchored output is not treated as publishable inline feedback.

## Publishing

When a native publisher is selected, DiffPal turns validated output into host
feedback such as a pull request summary, merge request discussion, file-level
comment, status, SARIF, or Code Quality report. The exact surfaces depend on
the code host and feedback mode.

## Artifact Generation

DiffPal writes machine-readable and human-readable artifacts under
`.artifacts/diffpal/`. These artifacts are useful for audit trails, downstream
CI steps, and supported report surfaces.

## Optional Gate

The gate is separate from feedback. When enabled, DiffPal compares findings to
the configured blocking threshold and returns a failing CI result if blocking
findings exist. When disabled, feedback and artifacts can still show blocking
findings without failing the merge check.

For exact config fields and exit behavior, use the
[configuration reference](../reference/configuration.md).
