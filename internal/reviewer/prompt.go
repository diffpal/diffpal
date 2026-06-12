package reviewer

import "strings"

const inputSchemaJSON = `{
  "type": "object",
  "properties": {
    "review_id": {"type": "string"},
    "repo": {"type": "string"},
    "base_sha": {"type": "string"},
    "head_sha": {"type": "string"},
    "chunk_index": {"type": "integer", "minimum": 0},
    "chunk_count": {"type": "integer", "minimum": 1},
    "language": {"type": "string"},
    "review_checks": {
      "type": "array",
      "items": {"type": "string", "enum": ["bugs", "performance", "best-practices"]}
    },
    "test_summary": {"type": "string"},
    "files": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "path": {"type": "string"},
          "signature": {"type": "string"},
          "snippet": {"type": "string"},
          "spans": {
            "type": "array",
            "items": {
              "type": "object",
              "properties": {
                "start": {"type": "integer", "minimum": 1},
                "end": {"type": "integer", "minimum": 1}
              },
              "required": ["start", "end"]
            }
          }
        },
        "required": ["path", "signature", "snippet", "spans"]
      }
    }
  },
  "required": ["review_id", "repo", "base_sha", "head_sha", "chunk_index", "chunk_count", "language", "review_checks", "test_summary", "files"]
}`

const outputSchemaJSON = `{
  "type": "object",
  "properties": {
    "change_summary": {
      "type": "array",
      "items": {"type": "string"},
      "maxItems": 8
    },
    "findings": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "rule_id": {"type": "string"},
          "category": {"type": "string", "enum": ["security", "correctness", "reliability", "performance", "maintainability", "testing", "style"]},
          "severity": {"type": "string", "enum": ["low", "medium", "high", "critical"]},
          "confidence": {"type": "number", "minimum": 0, "maximum": 1},
          "path": {"type": "string"},
          "start_line": {"type": "integer", "minimum": 1},
          "end_line": {"type": "integer", "minimum": 1},
          "title": {"type": "string"},
          "message": {"type": "string"},
          "evidence": {"type": "string"},
          "suggestion": {"type": "string"}
        },
        "required": ["rule_id", "category", "severity", "confidence", "path", "start_line", "end_line", "title", "message", "evidence"]
      }
    }
  },
  "required": ["change_summary", "findings"]
}`

func reviewInstruction() string {
	return strings.Join([]string{
		"You are DiffPal, an exhaustive code review agent.",
		"Review only the provided files and line spans.",
		"Use input.language for every finding title, message, evidence, and suggestion.",
		"Always return change_summary as concise bullets describing what changed in the provided diff chunk, even when there are no findings.",
		"Keep change_summary factual and based only on visible paths, snippets, signatures, and spans.",
		"Only run the requested input.review_checks.",
		"Map review_checks to categories as follows: bugs covers security, correctness, and reliability; performance covers performance; best-practices covers maintainability, testing, and style.",
		"Prefer high recall, but only report issues you can support with direct evidence from the provided snippets.",
		"Do not invent paths, line numbers, APIs, or behavior that are not visible in the input.",
		"Return one finding per distinct issue.",
		"Use rule_id format <category>.<slug>.",
		"Use critical/high only for severe actionable issues.",
		"If there are no issues, return an empty findings array.",
		"Suggestions are optional and should be short, concrete, and safe to apply.",
	}, "\n")
}
