# Support Matrix

DiffPal supports GitHub, GitLab, and Azure DevOps publishing targets.

| Host | Native outputs | Guide |
| --- | --- | --- |
| GitHub Actions | PR review summary, file-level review comments, SARIF | [GitHub Actions](../integrations/github-actions.md) |
| GitLab CI | MR summary, discussions, Code Quality, SARIF, status | [GitLab CI](../integrations/gitlab-ci.md) |
| Azure Pipelines | PR summary thread, PR threads, PR status | [Azure Pipelines](../integrations/azure-pipelines.md) |

The same `.config/diffpal/config.yaml` shape works across hosts. The CI file
changes how the provider is installed, how credentials are passed, and which
publisher DiffPal runs.
