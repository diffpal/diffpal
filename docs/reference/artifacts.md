# Artifacts Reference

DiffPal writes artifacts under `.artifacts/diffpal/` by default.

## Artifact Table

| Path | Created when | Format | Intended consumer |
| --- | --- | --- | --- |
| `.artifacts/diffpal/findings.json` | Every successful provider-backed review and `debug prompt` unless `--out` changes it. | DiffPal findings JSON | CI automation, audit logs, downstream converters. |
| `.artifacts/diffpal/summary.md` | Non-dry-run host review commands when the `summary` surface is enabled; custom CI examples may capture local stdout here. | Markdown | Humans, CI job summaries, retained review record. |
| `.artifacts/diffpal/diffpal.sarif` | Non-dry-run GitHub and GitLab host review commands; `diffpal sarif`; any file-publishing run that enables the SARIF surface. | SARIF 2.1.0 JSON | Code scanning or SARIF upload consumers. |
| `.artifacts/diffpal/codequality.json` | GitLab host review commands. | GitLab Code Quality JSON | GitLab Code Quality report ingestion. |
| `.artifacts/diffpal/github-comments.json` | GitHub file-output publishing plan when GitHub comment surface is emitted to files. | JSON | Debugging or API publishing handoff. |
| `.artifacts/diffpal/gitlab-discussions.json` | GitLab file-output publishing plan for discussions. | JSON | Debugging or API publishing handoff. |
| `.artifacts/diffpal/gitlab-status.json` | GitLab status file output. | JSON | Debugging or API publishing handoff. |
| `.artifacts/diffpal/azure-threads.json` | Azure file-output publishing plan for PR threads. | JSON | Debugging or API publishing handoff. |
| `.artifacts/diffpal/azure-status.json` | Azure status file output. | JSON | Debugging or API publishing handoff. |

## Canonical Versus Host-Specific

`findings.json` is the canonical DiffPal artifact. It contains validated
structured findings and metadata independent of a code host.

Host-specific artifacts are derived from the canonical bundle:

- SARIF is derived for SARIF consumers.
- GitLab Code Quality is derived for GitLab report ingestion.
- Comments, discussions, threads, and status JSON files are publishing plans or
  status payloads for host integrations.

## Creation Conditions

`diffpal review local`:

- writes `findings.json` or the `--out` path;
- prints Markdown to stdout;
- does not automatically create `summary.md`.

Non-dry-run `diffpal review github`:

- writes `findings.json`;
- creates `summary.md` and `diffpal.sarif`;
- creates GitHub comment artifacts when `feedback=review` file outputs are
  generated.

Non-dry-run `diffpal review gitlab`:

- writes `findings.json`;
- creates `summary.md`, `diffpal.sarif`, `codequality.json`,
  `gitlab-status.json`;
- creates `gitlab-discussions.json` when `feedback=review` file outputs are
  generated.

Non-dry-run `diffpal review ado`:

- writes `findings.json`;
- creates `summary.md` and `azure-status.json`;
- creates `azure-threads.json` when `feedback=review` file outputs are
  generated.

`diffpal debug prompt` writes a mock findings bundle to `--out`.
`diffpal sarif` reads a findings bundle and writes SARIF to `--out`.

## Retention Advice

Upload `.artifacts/diffpal/` from CI when you need audit records, downstream
automation, or report ingestion after the job completes. Retain at least
`findings.json` for machine processing and `summary.md` for human review
history when host summaries are not enough.
