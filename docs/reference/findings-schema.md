# Findings Schema

The canonical review artifact is
`.artifacts/diffpal/findings.json`, a JSON serialization of DiffPal's findings
bundle. New review writes use `version: "v5"` unless a bundle version is
already set by the caller. The canonical machine-readable contract is
[`schemas/findings/v5.schema.json`](../../schemas/findings/v5.schema.json);
LintPal keeps a byte-identical offline copy.

## Canonical Bundle

| Field | Required | Meaning |
| --- | --- | --- |
| `version` | yes | Bundle version. New writes use `v5`; readers accept `v1` through `v5`. |
| `review_id` | yes | Stable review identifier. |
| `base_sha` | yes | Base revision used for the review. |
| `head_sha` | yes | Head revision used for the review. |
| `merge_base_sha` | no | Resolved merge base, when available. |
| `language` | no | Language requested for generated text. |
| `prompt` | no | Prompt metadata. |
| `inspection` | no | Provider inspection metadata. |
| `change_summary` | no | Human-readable summary bullets. |
| `review_result` | no | Human-readable review outcome sentence. |
| `files` | no | Reviewed files. |
| `findings` | yes | Finding array. May be empty. |
| `skips` | no | Changed files skipped by the reviewer. |
| `stats` | no | Work and token counts. |

Prompt metadata fields:

| Field | Meaning |
| --- | --- |
| `prompt_id` | Prompt identifier, currently `diffpal.review` for review output. |
| `prompt_version` | Prompt version used for the review. |
| `purpose` | Prompt purpose. |
| `schema_version` | Prompt output schema version, currently `findings.v4`. |

The prompt schema version identifies provider output. It is independent of the
stored findings bundle version.

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
| `id` | yes for v5; written by DiffPal | Deterministic fingerprint. |
| `review_id` | yes for v5; written by DiffPal when missing | Review identifier copied from the bundle. |
| `category` | yes | Finding category. |
| `severity` | yes | `low`, `medium`, `high`, or `critical`. |
| `confidence` | for code evidence | Number from `0` to `1`. |
| `path` | yes | File path for the finding. |
| `start_line` | yes | Positive start line. |
| `end_line` | yes | Positive end line, greater than or equal to `start_line`. |
| `changed_span` | yes for v2 through v5 | Changed-line span that anchors the finding. |
| `supporting_span` | no | Additional context span. |
| `title` | yes | Short finding title. |
| `message` | yes | Finding explanation. |
| `evidence` | yes | `kind: "code"` with anchor, reasoning basis, and source; or `kind: "rule"` with Markdown `rule_id`. |
| `impact` | for code evidence | Structured impact. |
| `decision` | for rule evidence | `kind: "noul_probability"` and numeric `value` from `0` to `1`. |
| `suggestion` | no | Suggested fix. |
| `blocking` | yes for v5; written by DiffPal | Whether the finding meets the active threshold. |
| `provider` | yes for v5 | Provider ID that produced the finding. |
| `model` | no | Provider model when recorded. |
| `work_item_id` | no | LintPal work item identifier when recorded. |

Line span representation:

```json
{
  "path": "internal/session.go",
  "start_line": 12,
  "end_line": 14,
  "side": "RIGHT"
}
```

Evidence representation:

```json
{
  "kind": "code",
  "anchor": "changed lines call exec with request input",
  "reasoning_basis": "the command arguments now include unsanitized user data",
  "source": "changed_line"
}
```

For LintPal rule findings, evidence is a rule reference such as
`{"kind":"rule","rule_id":"go/errors.md"}`. The optional Markdown
frontmatter does not become evidence; it controls rule severity, title, and
decision threshold. Rule findings omit `confidence` and `impact`.

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

For v4 and v5, `changed_span.side` is also required. `LEFT` uses old-file line
coordinates for deleted lines; `RIGHT` uses new-file line coordinates for
added lines.

## Compatibility

DiffPal readers accept:

- `v1` bundles where `evidence` and `impact` may be legacy strings;
- `v2` bundles with structured evidence and impact;
- `v3` bundles with optional `review_result`;
- `v4` bundles with a side-aware changed-line anchor.
- `v5` bundles with code or rule evidence and a shared finding shape.

Legacy v1-v3 anchors default to `RIGHT` when read. New writes use `v5`.
Consumers should ignore unknown fields and treat
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
