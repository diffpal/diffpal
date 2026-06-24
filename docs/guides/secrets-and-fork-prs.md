# Secrets And Fork Pull Requests

Use this guide when you add provider credentials, host publishing tokens, or
fork pull request handling to a DiffPal workflow.

DiffPal runs in your CI and sends review input to the provider you configure.
When you use a remote provider, the reviewed diff and review instructions can
leave the CI job and be processed by that third-party provider.

## What Receives Repository Content

The CI job checks out the repository working tree and runs DiffPal in that
workspace. DiffPal reads the configured `base..head` diff, repository-owned
configuration, and review instructions.

The selected provider or ACP-compatible agent receives the review request that
DiffPal builds from the changed files. For remote providers, that request is
sent outside the CI runner according to the provider's own service terms and
account settings.

Host publishers receive validated review output. GitHub, GitLab, and Azure
DevOps receive summaries, comments, discussions, threads, statuses, or reports
only when you run a publishing command or action with host credentials.

## Credential Types

Provider credentials authenticate the AI provider or agent. Examples include:

| Credential | Belongs to |
| --- | --- |
| `OPENAI_API_KEY` | Codex API-key provider authentication |
| `CODEX_AUTH_JSON_B64` | Codex subscription auth restored inside CI |
| `COPILOT_GITHUB_TOKEN` | Copilot CLI provider authentication |
| Custom provider secrets | The ACP-compatible CLI or provider you configure |

Host credentials authenticate the code host publisher. Examples include:

| Credential | Belongs to |
| --- | --- |
| `GITHUB_TOKEN` | GitHub pull request review and comments |
| `CI_JOB_TOKEN` or `GITLAB_TOKEN` | GitLab merge request discussions, status, and reports |
| `SYSTEM_ACCESSTOKEN` or `AZURE_DEVOPS_EXT_PAT` | Azure DevOps PR threads and status |

Keep provider credentials separate from host credentials so each secret can use
the least privilege needed for its own system.

## Minimum Host Permissions

Use the smallest host permission set that supports the configured feedback
mode.

| Host | Minimum verified permissions |
| --- | --- |
| GitHub | `contents: read` and `pull-requests: write` for PR summaries and review comments. |
| GitLab | A job token or API token that can read the project, create merge request discussions, and publish commit status. Code Quality and SARIF reports are uploaded by the CI artifacts mechanism. |
| Azure DevOps | Pipeline OAuth access through `System.AccessToken` for PR threads and status, or a dedicated PAT with equivalent PR thread/status permission when required by the organization. |

Artifact-only `diffpal review local` does not need a host publishing token, but
it still needs provider credentials when the configured provider is remote.

## Protected And Masked Secrets

Store provider and host credentials as CI secrets or protected variables. Mark
them masked when the CI system supports masking, and restrict them to trusted
branches, protected environments, same-repository pull requests, or
maintainer-approved jobs.

Do not commit provider credentials to `.config/diffpal/config.yaml`, workflow
files, examples, artifacts, or issue comments.

## Fork Pull Request Safety

Untrusted fork pipelines must not receive provider credentials. A fork author
can change workflow files, package scripts, build hooks, provider commands, or
test code in ways that run before or during the review job.

Safe patterns for external contributions:

- Run normal no-secret CI on fork pull requests.
- Run DiffPal with provider credentials only on same-repository pull requests.
- Use a maintainer-approved job that does not execute fork-controlled code with
  secrets.
- Review workflow changes before manually re-running any credentialed job.
- Keep `pull_request_target` jobs away from PR-head checkout, package installs,
  build scripts, hooks, provider CLIs, and other fork-controlled execution.

GitHub Actions workflows should keep a same-repository guard next to the
credentialed review job:

```yaml
if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
```

GitLab CI can combine same-project merge request conditions, protected
variables, and a manual maintainer gate such as `DIFFPAL_TRUSTED_REVIEW`.

Azure Pipelines can skip credentialed review for fork validation with:

```yaml
condition: and(succeeded(), ne(variables['System.PullRequest.IsFork'], 'True'))
```

## Artifact Retention

DiffPal artifacts can contain file paths, line numbers, findings, summaries,
and snippets or descriptions derived from reviewed code. Treat
`.artifacts/diffpal/` as repository-sensitive output.

Retain artifacts long enough for audit, debugging, SARIF ingestion, or Code
Quality ingestion. Avoid uploading them to public storage from private
repositories, and apply the same retention rules you use for CI logs and test
reports.

## Log Redaction

CI systems usually mask exact secret values, but masking is not a complete
redaction boundary. Avoid `set -x`, do not echo tokens, and do not print
restored provider auth files. Be careful with derived values, encoded secrets,
provider debug logs, and command-line arguments because they may not match the
secret value the CI system knows how to mask.

If you need provider debugging, use the provider's own safe logging controls
and remove sensitive logs from retained artifacts.

## Report A Security Issue

This repository does not currently include a `SECURITY.md` policy file. If the
GitHub repository exposes **Report a vulnerability** on the Security tab, use
that private channel. Otherwise, open a GitHub issue with only a minimal,
non-sensitive description and ask maintainers for a private reporting channel.

Do not paste secrets, exploit details, private code, provider responses, or
private artifacts into a public issue.

See the public [Security controls](../security.md) page for the shorter control
summary.

Next step: apply the trusted-source guard from your host integration page before
enabling provider credentials in CI.
