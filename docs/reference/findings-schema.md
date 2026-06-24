# Findings Schema

The canonical review artifact is
`.artifacts/diffpal/findings.json`, a JSON serialization of DiffPal's findings
bundle. New review writes use `version: "v3"` unless a bundle version is
already set by the caller.

## Canonical Bundle

| Field | Required | Meaning |
| --- | --- | --- |
| `version` | yes | Bundle version. New writes use `v3`; readers accept `v1`, `v2`, and `v3`. |
| `review_id` | yes | Stable review identifier. |
| `base_sha` | yes | Base revision used for the review. |
| `head_sha` | yes | Head revision used for the review. |
| `language` | no | Language requested for generated text. |
| `prompt` | no | Prompt metadata. |
| `inspection` | no | Provider inspection metadata. |
| `change_summary` | no | Human-readable summary bullets. |
| `review_result` | no | Human-readable review outcome sentence. |
| `files` | no | Reviewed files. |
| `findings` | yes | Finding array. May be empty. |

Prompt metadata fields:

| Field | Meaning |
| --- | --- |
| `prompt_id` | Prompt identifier, currently `diffpal.review` for review output. |
| `prompt_version` | Prompt version used for the review. |
| `purpose` | Prompt purpose. |
| `schema_version` | Prompt output schema version, currently `findings.v3`. |

Inspection metadata fields:

| Field | Meaning |
| --- | --- |
| `provider_type` | Runtime provider type when available. |
| `required` | Whether inspection metadata was required by the runtime. |
| `tool_calls` | Tool call names when available. |
| `diff_inspected` | Whether the provider reported diff inspection. |
| `context_inspected` | Whether the provider reported context inspection. |

## Finding Fields

| Field | Required | Meaning |
| --- | --- | --- |
| `id` | written by DiffPal | Deterministic fingerprint. |
| `review_id` | written by DiffPal when missing | Review identifier copied from the bundle. |
| `category` | yes | Finding category. |
| `severity` | yes | `low`, `medium`, `high`, or `critical`. |
| `confidence` | yes | Number from `0` to `1`. |
| `path` | yes | File path for the finding. |
| `start_line` | yes | Positive start line. |
| `end_line` | yes | Positive end line, greater than or equal to `start_line`. |
| `changed_span` | yes for v2/v3 | Changed-line span that anchors the finding. |
| `supporting_span` | no | Additional context span. |
| `title` | yes | Short finding title. |
| `message` | yes | Finding explanation. |
| `evidence` | yes | Structured evidence for v2/v3. |
| `impact` | yes | Structured impact for v2/v3. |
| `suggestion` | no | Suggested fix. |
| `blocking` | written by DiffPal | Whether the finding meets the active threshold. |
| `provider` | no | Provider ID that produced the finding. |

Line span representation:

```json
{
  "path": "internal/session.go",
  "start_line": 12,
  "end_line": 14
}
```

Evidence representation:

```json
{
  "anchor": "changed lines call exec with request input",
  "reasoning_basis": "the command arguments now include unsanitized user data",
  "source": "changed_line"
}
```

Impact representation:

```json
{
  "summary": "users can execute unintended shell commands",
  "scope": "request handling path"
}
```

## Severity And Location

Allowed severities are `low`, `medium`, `high`, and `critical`.
DiffPal normalizes severity to lowercase.

Location is represented twice for compatibility:

- `path`, `start_line`, and `end_line` are the primary line fields;
- `changed_span` carries the same changed-line anchor in structured form.

For v2/v3, `changed_span.path`, `changed_span.start_line`, and
`changed_span.end_line` are required and must be positive.

## Compatibility

DiffPal readers accept:

- `v1` bundles where `evidence` and `impact` may be legacy strings;
- `v2` bundles with structured evidence and impact;
- `v3` bundles with optional `review_result`.

New writes use `v3`. Consumers should ignore unknown fields and treat
`findings[]` as the canonical machine-readable issue list.

## Consumer Example

Fail a CI step when the canonical bundle contains blocking findings:

```bash
jq -e '[.findings[] | select(.blocking == true)] | length == 0' \
  .artifacts/diffpal/findings.json
```

Count high and critical findings regardless of whether the gate was enabled:

```bash
jq '[.findings[] | select(.severity == "high" or .severity == "critical")] | length' \
  .artifacts/diffpal/findings.json
```
