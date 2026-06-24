# Support Matrix

This page lists the supported public surface confirmed by the repository.

## Code Hosts And Publishers

| Host / mode | Native publisher | Stable status | Guide |
| --- | --- | --- | --- |
| GitHub pull requests | GitHub publisher | Stable public surface | [GitHub Actions](../integrations/github-actions.md) |
| GitLab merge requests | GitLab publisher | Stable public surface | [GitLab CI](../integrations/gitlab-ci.md) |
| Azure DevOps pull requests | Azure DevOps publisher | Stable public surface | [Azure Pipelines](../integrations/azure-pipelines.md) |
| Custom CI/CD with unsupported code host | No native publisher | Artifact-only support | [Custom CI/CD](../integrations/custom-ci.md) |

## Feedback Surfaces

| Host | `summary` feedback | `review` feedback |
| --- | --- | --- |
| GitHub | Summary and SARIF | Summary, file-level PR review comments, SARIF |
| GitLab | Summary, status, SARIF, Code Quality | Summary, discussions, status, SARIF, Code Quality |
| Azure DevOps | Summary and status | Summary, PR threads, status |
| Custom CI/CD unsupported host | Local Markdown and artifacts | Local Markdown and artifacts |

## Artifact Surfaces

| Surface | GitHub | GitLab | Azure | Local/custom |
| --- | --- | --- | --- | --- |
| Findings bundle | yes | yes | yes | yes |
| Summary Markdown | yes | yes | yes | stdout unless captured |
| SARIF | yes | yes | no native surface | via `diffpal sarif` |
| GitLab Code Quality | no | yes | no | no |
| Host status | GitHub check/workflow status | GitLab commit status | Azure PR status | CI job status |

## Provider Types

Supported runtime provider types:

| Type | Status | Notes |
| --- | --- | --- |
| `codex_acp` | Stable public setup | Default onboarding provider type. |
| `copilot_acp` | Stable public setup | Documented provider page and examples. |
| `opencode_acp` | Stable public setup | Documented provider page and config example. |
| `generic_acp` | Stable public setup | Bring your own ACP-compatible CLI. |
| `openai` | Supported config type | Hosted API config type; use when explicitly configured. |
| `aistudio` | Supported config type | Hosted API config type; use when explicitly configured. |
| `gemini_acp` | Supported runtime type | No dedicated public setup page in this docs set. |
| `claude_code_acp` | Supported runtime type | No dedicated public setup page in this docs set. |
| `pool` | Supported runtime type | Ordered provider failover config. |

No provider account is managed by DiffPal. Provider authentication belongs to
the selected provider and CI secret management.
