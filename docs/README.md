# DiffPal Docs

Choose the path that matches what you want to do first.

## I Want The Fastest GitHub Setup

Use the [quickstart](quickstart.md) to generate a config, add one provider
secret, copy a GitHub Actions workflow, and get the first PR review running.

This path uses the Codex API-key recipe because it is a ready-made example. You
can switch providers later without changing the DiffPal review workflow.

## I Already Use Another Agent

Start with [Using Another ACP CLI](ci-examples.md#using-another-acp-cli). Keep
your existing CLI, account, tools, and provider-specific authentication, then
point `runtime.providers` at the command that starts its ACP stdio server.

## I Need GitLab Or Azure DevOps

Use the [CI setup guide](ci-examples.md) for host-specific checkout, token, and
publishing requirements. The same committed DiffPal config shape works across
GitHub, GitLab, and Azure DevOps.

## I Want Stricter Policy And Auditing

Read the [config reference](config-reference.md), [findings schema](findings-schema.md),
and [what success looks like](what-success-looks-like.md). DiffPal records prompt
metadata, writes a structured findings bundle, and can fail CI when findings
meet your `block_on` threshold.

## Common Next Steps

- [Examples gallery](../examples/README.md) for copy-paste configs and CI files
- [Comparison guide](comparison.md) for how DiffPal differs from hosted
  reviewers and lint publishers
- [Troubleshooting](troubleshooting.md) for missing comments, token failures,
  fork PRs, and incomplete diffs
- [Visual assets plan](visual-assets.md) for screenshots, diagrams, and demo
  assets maintainers should produce
