# Findings Schema v1

Canonical output is `internal/findings.FindingsBundle` serialized to JSON.

Required fields:

- `version`
- `review_id`
- `base_sha`
- `head_sha`
- `language` (optional)
- `review_checks[]` (optional)
- `change_summary[]` (optional human-readable overview bullets)
- `files[]` (optional reviewed file list)
- `findings[]` with per-item fields:
  - `category`
  - `severity`
  - `path`
  - `start_line`
  - `end_line`
  - `title`
  - `message`
  - `evidence`

Stable fingerprint input:

- repository id
- `platform` (`diffpal`)
- `review_id`
- `head_sha`
- normalized path and line range
- category
- normalized message
- evidence hash

`findings.Normalize` computes `finding.id` deterministically from fields above.
