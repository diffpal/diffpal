# Findings Schema v1

Canonical output is `internal/findings.FindingsBundle` serialized to JSON.

Required fields:

- `version`
- `review_id`
- `base_sha`
- `head_sha`
- `findings[]` with per-item fields:
  - `rule_id`
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
- normalized message
- evidence hash

`findings.Normalize` computes `finding.id` deterministically from fields above.
