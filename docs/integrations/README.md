# Integrations

Use this section to run DiffPal in CI and publish review feedback to your code
host. Host-specific pages all follow the same shape:

For the review flow behind these CI steps, see
[Review lifecycle](../concepts/review-lifecycle.md).

- [GitHub Actions](github-actions.md)
- [GitLab CI](gitlab-ci.md)
- [Azure Pipelines](azure-pipelines.md)
- [Custom CI/CD](custom-ci.md)

Copy-paste configs and pipelines live in [`examples/`](../../examples/README.md).
Use the [GitHub quickstart](../getting-started/github-quickstart.md) when you
want the shortest first setup path.
Use [Providers](../providers/README.md) to choose Codex, Copilot, OpenCode, or
a custom ACP-compatible CLI.

## Shared Setup

Every host needs:

1. Full git history for the reviewed pull request or merge request.
2. A DiffPal config committed at `.config/diffpal/config.yaml`.
3. The provider CLI runtime required by the selected
   [provider](../providers/README.md).
4. A provider auth secret.
5. A platform token with permission to publish review feedback.

DiffPal runs in your CI and sends review input to the provider you configure.
Protect provider credentials before enabling secret-backed review. See
[Secrets and fork PRs](../guides/secrets-and-fork-prs.md).

For Jenkins, Buildkite, CircleCI, Bitbucket Pipelines, internal runners, or any
other CI system, use the [Custom CI/CD guide](custom-ci.md).

## Feedback Modes

Use `feedback` for normal setup:

| Feedback | Behavior |
| --- | --- |
| `summary` | PR/MR summary plus non-file artifacts such as status, SARIF, or Code Quality. No file-level findings are published. |
| `review` | Summary plus file-level comments, threads, or discussions for the platform. Non-blocking findings remain visible without becoming merge blockers. |

Default review publish surfaces:

| Platform | Default surfaces |
| --- | --- |
| GitHub | `comments,sarif,summary` |
| GitLab | `code-quality,discussions,status,sarif,summary` |
| Azure | `threads,status,summary` |

Common artifacts are listed in the [artifacts reference](../reference/artifacts.md).

## Merge Gates

Enable `gate` when blocking findings should fail the CI job. Start with
`block_on: high`; lower the threshold only after tuning review policy. See the
[configuration gate reference](../reference/configuration.md#gate) and
[exit behavior](../reference/exit-behavior.md).

Tooling failures such as setup, provider auth, review scope resolution, or
publishing fail the job because the review result is incomplete, even when the
merge gate is disabled.

## Untrusted Contributions

Keep provider credentials out of untrusted fork pipelines. Run secret-backed
DiffPal review only for trusted branches, same-repository pull requests, or
maintainer-approved workflows that do not execute untrusted code with secrets.

See [Secrets and fork PRs](../guides/secrets-and-fork-prs.md).

## Common Failures

Most integration failures come from:

- shallow checkout;
- missing provider secret;
- provider CLI not installed or authenticated;
- platform token missing write permission;
- running secret-backed review on an untrusted fork PR.

Use the [troubleshooting guide](../help/troubleshooting.md) for fixes.

Next step: open the host-specific integration page for the CI system that will
run DiffPal.
