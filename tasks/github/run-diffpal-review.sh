#!/usr/bin/env bash
set -euo pipefail

diffpal_bin="${DIFFPAL_BIN:-diffpal}"

if ! command -v "$diffpal_bin" >/dev/null 2>&1; then
  echo "diffpal binary was not found: $diffpal_bin" >&2
  exit 127
fi

require_input() {
  local name="$1"
  local value="$2"
  if [[ -z "$value" ]]; then
    echo "required input is empty: $name" >&2
    exit 2
  fi
}

truthy() {
  case "${1,,}" in
    1 | true | yes | y | on)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

require_input "base" "${INPUT_BASE:-}"
require_input "head" "${INPUT_HEAD:-}"

argv=("$diffpal_bin")

if [[ -n "${INPUT_CONFIG_DIR:-}" ]]; then
  argv+=(--config-dir "$INPUT_CONFIG_DIR")
fi
if [[ -n "${INPUT_PROFILE:-}" ]]; then
  argv+=(--profile "$INPUT_PROFILE")
fi

argv+=(review github --base "$INPUT_BASE" --head "$INPUT_HEAD")

if [[ -n "${INPUT_BLOCK_ON:-}" ]]; then
  argv+=(--block-on "$INPUT_BLOCK_ON")
fi
if truthy "${INPUT_GATE:-false}"; then
  argv+=(--gate)
fi
if [[ -n "${INPUT_MODE:-}" ]]; then
  argv+=(--mode "$INPUT_MODE")
fi
if [[ -z "${INPUT_MODE:-}" && -n "${INPUT_FEEDBACK:-}" ]]; then
  argv+=(--feedback "$INPUT_FEEDBACK")
fi
if ! truthy "${INPUT_SUMMARY_OVERVIEW:-true}"; then
  argv+=(--summary-overview=false)
fi
if [[ -n "${INPUT_OUT:-}" ]]; then
  argv+=(--out "$INPUT_OUT")
fi
if [[ -n "${INPUT_REPO:-}" ]]; then
  argv+=(--repo "$INPUT_REPO")
fi
if [[ -n "${INPUT_REVIEW_ID:-}" ]]; then
  argv+=(--review-id "$INPUT_REVIEW_ID")
fi
if [[ -n "${INPUT_REVIEW_CHANNEL:-}" ]]; then
  argv+=(--review-channel "$INPUT_REVIEW_CHANNEL")
fi
if [[ -n "${INPUT_MAX_FILES:-}" ]]; then
  argv+=(--max-files "$INPUT_MAX_FILES")
fi
if [[ -n "${INPUT_CONTEXT_LINES:-}" ]]; then
  argv+=(--context-lines "$INPUT_CONTEXT_LINES")
fi
if [[ -n "${INPUT_MAX_PATCH_CHARS:-}" ]]; then
  argv+=(--max-patch-chars "$INPUT_MAX_PATCH_CHARS")
fi
if [[ -n "${INPUT_MAX_FILES_PER_CHUNK:-}" ]]; then
  argv+=(--max-files-per-chunk "$INPUT_MAX_FILES_PER_CHUNK")
fi
if [[ -n "${INPUT_LANGUAGE:-}" ]]; then
  argv+=(--language "$INPUT_LANGUAGE")
fi
if [[ -n "${INPUT_REVIEW_CHECKS:-}" ]]; then
  argv+=(--review-checks "$INPUT_REVIEW_CHECKS")
fi
if [[ -n "${INPUT_INSTRUCTIONS:-}" ]]; then
  argv+=(--instructions "$INPUT_INSTRUCTIONS")
fi
if [[ -n "${INPUT_INSTRUCTIONS_FILE:-}" ]]; then
  argv+=(--instructions-file "$INPUT_INSTRUCTIONS_FILE")
fi

exec "${argv[@]}"
