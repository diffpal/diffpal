# Security Controls

DiffPal runs in your CI and sends review input to the provider you configure.
This page summarizes the public security controls users should understand
before enabling provider-backed review.

## Boundaries

- The CI runner receives the repository checkout, DiffPal configuration, and
  provider credentials made available to the job.
- DiffPal reads the configured diff and sends review input to the selected
  provider or ACP-compatible agent.
- The provider owns model execution, provider account security, and provider
  credentials.
- GitHub, GitLab, and Azure DevOps publishers receive review output only when a
  publishing command or action runs with host credentials.

## Credential Controls

Keep provider credentials and host credentials separate.

Provider credentials include values such as `OPENAI_API_KEY`,
`CODEX_AUTH_JSON_B64`, `COPILOT_GITHUB_TOKEN`, and custom ACP provider tokens.
Host credentials include `GITHUB_TOKEN`, `CI_JOB_TOKEN`, `GITLAB_TOKEN`,
`SYSTEM_ACCESSTOKEN`, and `AZURE_DEVOPS_EXT_PAT`.

Store credentials as protected and masked CI secrets. Give host tokens only the
minimum permission needed to publish the selected feedback mode.

## Fork Pull Requests

Do not expose provider or host credentials to untrusted fork code. Run
secret-backed DiffPal review only for same-repository pull requests, trusted
branches, or maintainer-approved jobs that do not execute fork-controlled code
with secrets.

Use [Secrets and fork PRs](guides/secrets-and-fork-prs.md) for host-specific
guards and safe external contribution patterns.

## Artifacts And Logs

DiffPal artifacts and logs can contain repository-sensitive review output.
Retain `.artifacts/diffpal/` only where you need audit records, SARIF or Code
Quality ingestion, or troubleshooting. Do not rely on CI masking as the only
protection for debug logs or restored provider auth files.

## Reporting Security Issues

This repository does not currently include a `SECURITY.md` policy file. If the
GitHub repository exposes **Report a vulnerability** on the Security tab, use
that private channel. Otherwise, open a GitHub issue with a minimal,
non-sensitive description and ask maintainers for a private reporting channel.

Do not include secrets, exploit details, private code, provider responses, or
private artifacts in a public issue.

