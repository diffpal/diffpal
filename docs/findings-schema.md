# Findings Schema v3

Canonical output is `internal/findings.FindingsBundle` serialized to JSON.
New DiffPal review runs write `version: v3` and prompt metadata
`schema_version: findings.v3`.

Required top-level fields:

- `version`
- `review_id`
- `base_sha`
- `head_sha`
- `findings[]`

Optional top-level fields:

- `language`
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
- `review_result` optional human-readable review outcome sentence
- `files[]` reviewed file list

## Prompt Versioning

Prompt metadata is resolved from the versioned prompt registry in
`internal/reviewer/promptpack`. The current default review prompt is:

- `prompt_id`: `diffpal.review`
- `prompt_version`: `v1.4.0`
- `purpose`: `review_changed_diff`
- `schema_version`: `findings.v3`

Prompt body, output schema, and task instructions are treated as a versioned
product surface. When changing the prompt contract, add a new registered prompt
version, update prompt snapshot tests, document the schema impact here, and add
a changelog entry. Do not silently change `prompt_version` metadata without a
matching prompt registry and golden-test update.

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

Providers inspect the requested base/head diff and supporting code through their
available Git and filesystem tooling. DiffPal no longer requires provider-facing
review tool telemetry before accepting structured output.

The `inspection` object remains in the bundle schema for compatibility with
older bundles and debug runtimes that can still emit inspection metadata.

## Compatibility

DiffPal can still read existing `version: v1` bundles where `evidence` and
`impact` are strings, plus `version: v2` bundles without `review_result`.
New writes use `version: v3`; prompt output validation requires structured
`evidence` and `impact`, accepts optional `review_result`, and rejects
unexpected provider properties.
