# Findings Schema v2

Canonical output is `internal/findings.FindingsBundle` serialized to JSON.
New DiffPal review runs write `version: v2` and prompt metadata
`schema_version: findings.v2`.

Required top-level fields:

- `version`
- `review_id`
- `base_sha`
- `head_sha`
- `findings[]`

Optional top-level fields:

- `language`
- `review_checks[]`
- `prompt` prompt pack metadata:
  - `prompt_id`
  - `prompt_version`
  - `purpose`
  - `schema_version`
- `inspection` provider inspection metadata:
  - `provider_type`
  - `required`
  - `tool_calls[]`
  - `diff_inspected`
  - `context_inspected`
- `change_summary[]` human-readable overview bullets
- `files[]` reviewed file list

Required finding fields:

- `category`
- `severity`
- `confidence`
- `path`
- `start_line`
- `end_line`
- `changed_span`
- `title`
- `message`
- `evidence`
- `impact`

`changed_span` is the changed diff range that anchors the finding:

```json
{
  "path": "app/session.go",
  "start_line": 12,
  "end_line": 13
}
```

`supporting_span` is optional nearby context. It must not replace the changed
line anchor.

`evidence` is structured:

```json
{
  "anchor": "L12-L13",
  "reasoning_basis": "the changed lines concatenate request input into SQL",
  "source": "changed_line"
}
```

Allowed `evidence.source` values are:

- `changed_line`
- `nearby_context`
- `tool_result`

`impact` is structured:

```json
{
  "summary": "attackers can delete unrelated sessions",
  "scope": "authenticated sessions"
}
```

Optional finding fields:

- `supporting_span`
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
- structured evidence text hash

`findings.Normalize` computes `finding.id` deterministically from fields above.

## Inspection Metadata

Hosted providers receive DiffPal review tools at runtime. A hosted review result
is accepted only after `git_diff` was called for the reviewed chunk. Tool usage
is recorded in `inspection.tool_calls`, with `diff_inspected` set when
`git_diff` ran.

ACP providers use their native tool surface. DiffPal records
`inspection.required: false` for those providers because runtime-level proof is
not available from the wrapper.

## Compatibility

DiffPal can still read existing `version: v1` bundles where `evidence` and
`impact` are strings. New writes use `version: v2`; prompt output validation
requires structured `evidence` and `impact` and rejects unexpected provider
properties.
