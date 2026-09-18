# Setup provider guidance

Read this reference only when adding, replacing, or repairing provider configuration. Prefer checked-in provider documentation and examples when the target repository contains them.

## Provider selection

| Type | Expected runtime | Configuration and authentication boundary |
| --- | --- | --- |
| `codex_acp` | `codex` plus its ACP bridge | Wizard setups: `codex-api-key` or `codex-subscription`. Authentication belongs to Codex; use `OPENAI_API_KEY` or an existing trusted subscription login as documented. |
| `copilot_acp` | `copilot` | Wizard setup: `copilot-github-token`. In CI prefer a separate `COPILOT_GITHUB_TOKEN`, not the host publishing token. |
| `opencode_acp` | `opencode` | Wizard setup: `opencode-acp`. Installation, model availability, and authentication belong to OpenCode. |
| `registry_acp` | `npx -y @baldaworks/acprun@<bridge_version> run <registry_id>` or the first argv entry from `cmd` | No wizard recipe. Require an operator-selected official registry ID or an explicit pinned command. Registry inclusion is not a compatibility or security endorsement. |
| `generic_acp` | First argv entry from `generic_acp.cmd` | Wizard setup: `generic-acp`. Require the user or repository to supply the exact noninteractive ACP stdio command. |
| `gemini_acp` | Deprecated and rejected by the shared runtime | Use `generic_acp` with an explicitly verified Gemini ACP command when required. |
| `claude_code_acp` | `claude` unless `cmd` overrides it | No wizard recipe. Preserve an existing verified block or require authoritative command/model/auth details. |
| `openai` | Hosted API; no provider CLI | Require a model and `OPENAI_API_KEY` from the environment or secret store. Do not write the value into YAML. |
| `aistudio` | Hosted API; no provider CLI | Require a model and `GEMINI_API_KEY` from the environment or secret store. Do not write the value into YAML. |
| `pool` | Its referenced providers | Validate every member ID, order, and underlying provider independently. Do not create empty, cyclic, or unknown members. |

Provider IDs are repository-owned names. Reuse a clear existing ID when repairing it; do not rename IDs and all consumers unless requested.

## Configuration invariants

- Define providers under root `runtime.providers`; profiles select them through `diffpal.provider`.
- The provider type must match its configuration block.
- Preserve `cmd`, `extra_args`, `model`, `model_config_id`, `mode`, `reasoning_effort`, `reasoning_effort_config_id`, `bridge_version`, `registry_id`, `system_instructions`, and `mcp_servers` unless the requested change targets them.
- Validate referenced MCP server IDs without exposing their `env` or header values.
- Treat provider package versions, models, commands, and authentication flows as provider-specific facts. Use repository evidence or ask for the missing choice rather than guessing.
- `doctor --mode local` validates structure and executable presence. It does not call the provider and may not prove account authentication.

## Credential handling

Refer only to credential names and documented placeholders. Do not read, print, transfer, or embed values. Provider authentication authorizes repository-content processing; GitHub, GitLab, or Azure authentication separately authorizes publishing.
