# Findings Schema v1

Canonical output is `internal/findings.FindingsBundle` serialized to JSON.

Required top-level fields:

- `version`
- `review_id`
- `base_sha`
- `head_sha`
- `findings[]` with per-item fields:
  - `category`
  - `severity`
  - `path`
  - `start_line`
  - `end_line`
  - `title`
  - `message`
  - `evidence`

Optional top-level fields:

- `language`
- `review_checks[]`
- `change_summary[]` human-readable overview bullets
- `files[]` reviewed file list

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
