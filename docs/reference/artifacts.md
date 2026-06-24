# Artifacts

DiffPal writes review artifacts under `.artifacts/diffpal/` by default. Use
these files for auditing, CI uploads, SARIF ingestion, and host-specific report
surfaces.

| Path | Purpose |
| --- | --- |
| `.artifacts/diffpal/findings.json` | Canonical structured findings bundle. |
| `.artifacts/diffpal/summary.md` | Human-readable review summary. |
| `.artifacts/diffpal/diffpal.sarif` | SARIF report when enabled by the platform output. |
| `.artifacts/diffpal/codequality.json` | GitLab Code Quality report. |

The findings bundle includes prompt metadata such as prompt id, prompt version,
purpose, and findings schema version. See the
[findings schema](findings-schema.md) for the bundle contract.
