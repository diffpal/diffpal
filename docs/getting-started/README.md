# Getting Started

Use this section to choose the shortest path to your first useful DiffPal
review.

## Prerequisites

Before you start, have:

- a repository on GitHub, GitLab, or Azure DevOps;
- permission to add CI configuration and repository secrets;
- one supported review provider or ACP-compatible agent;
- a pull request or merge request you can use for a trusted first test.

If the terms are new, read [How DiffPal works](../concepts/how-diffpal-works.md)
and the [Glossary](../concepts/glossary.md) first.

## Choose Code Host

| Host | Start here |
| --- | --- |
| GitHub Actions | [GitHub quickstart](github-quickstart.md) |
| GitLab CI | [GitLab CI guide](../integrations/gitlab-ci.md) |
| Azure Pipelines | [Azure Pipelines guide](../integrations/azure-pipelines.md) |
| Custom CI/CD | [Custom CI/CD guide](../integrations/custom-ci.md) |

## Choose Provider

The fastest GitHub path uses the Codex API-key recipe because it is ready to
copy. Codex is not the product boundary: the same DiffPal workflow works with
other supported providers and ACP-compatible agents.

| Provider path | Start here |
| --- | --- |
| Codex API key | [GitHub quickstart](github-quickstart.md) |
| Codex subscription auth, Copilot, OpenCode, or generic ACP | [Providers](../providers/README.md) |

## Choose Feedback Mode

| Mode | Use when |
| --- | --- |
| `summary` | You want a PR/MR summary and non-file artifacts without inline review comments. |
| `review` | You want the summary plus file-level comments, discussions, or PR threads. |

The GitHub quickstart uses `review` so the first run shows the full review
surface.

## Start

Use [GitHub quickstart](github-quickstart.md) to install DiffPal in a new GitHub
repository, then use [Verify First Review](verify-first-review.md) to confirm
the first run worked. After that, continue with [Next Steps](next-steps.md).
