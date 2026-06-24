# GitHub Actions

Use the [GitHub quickstart](../getting-started/github-quickstart.md) for the
fastest setup. This page summarizes the GitHub Actions requirements for custom
or adapted workflows.

Examples:

- [Codex API key](../../examples/ci/github-actions/codex-api-key.yml)
- [Codex subscription auth](../../examples/ci/github-actions/codex-subscription.yml)
- [Copilot token](../../examples/ci/github-actions/copilot-github-token.yml)

Required permissions:

```yaml
permissions:
  contents: read
  pull-requests: write
```

Use a same-repository PR guard before exposing provider secrets:

```yaml
if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
```

What you should see:

- A PR review headed `DiffPal Review Summary`.
- Inline review comments when DiffPal finds actionable issues.
- Job failure only when `gate` is set and blocking findings exist, or when setup
  or publish fails.

For provider setup and feedback modes, see the
[integrations guide](README.md). For expected results, see
[Verify First Review](../getting-started/verify-first-review.md).
