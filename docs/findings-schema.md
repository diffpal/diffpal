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
  - `confidence`
  - `path`
  - `start_line`
  - `end_line`
  - `title`
  - `message`
  - `evidence`
  - `impact`

Optional top-level fields:

- `language`
- `review_checks[]`
- `prompt` prompt pack metadata:
  - `prompt_id`
  - `prompt_version`
  - `purpose`
  - `schema_version`
- `change_summary[]` human-readable overview bullets
- `files[]` reviewed file list

Optional finding fields:

- `suggestion`
- `blocking`
- `provider`

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
