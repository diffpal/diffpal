# Findings, Feedback, And Gates

DiffPal separates the review result from where it is shown and whether it
blocks a merge.

## Finding

A finding is one concrete issue returned by the provider and validated by
DiffPal. It includes a category, severity, message, impact, and changed-file
location when the issue can be anchored inline.

## Severity

Severity describes concrete impact, not confidence or preference. DiffPal uses
severity to decide which findings are advisory and which meet the configured
blocking threshold.

## Summary

The summary is the top-level human-readable review output. It explains the
reviewed change and the review result even when there are no actionable
findings.

## Inline Feedback

Inline feedback is file-level feedback on changed lines, such as GitHub review
comments, GitLab discussions, or Azure PR threads. Inline feedback depends on
the selected host publisher and feedback mode.

## Output Artifact

An output artifact is a file written by DiffPal, such as
`.artifacts/diffpal/findings.json`, `summary.md`, SARIF, or GitLab Code Quality
JSON. Artifacts let CI retain the review result even when no inline comments
are published.

## Blocking Threshold

The blocking threshold is the minimum severity that counts as blocking. A
repository might start with `high` so only high and critical findings block the
gate.

## Feedback And Merge Blocking Are Separate

Feedback visibility answers "what should users see?" Merge blocking answers
"should this CI job fail?" Keeping them separate lets teams roll out DiffPal
safely:

- publish summaries and artifacts without inline comments;
- show inline feedback without failing merges;
- enable a gate only after the team agrees on the blocking threshold;
- lower the threshold later without changing provider setup.

Use [Integrations](../integrations/README.md) for host feedback modes and
[Configuration](../reference/configuration.md#gate) for the gate field.
