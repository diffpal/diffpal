# DiffPal

[![ci](https://github.com/diffpal/diffpal/actions/workflows/ci.yml/badge.svg)](https://github.com/diffpal/diffpal/actions/workflows/ci.yml)
[![diffpal-dev review](https://github.com/diffpal/diffpal/actions/workflows/diffpal-dev-review.yml/badge.svg)](https://github.com/diffpal/diffpal/actions/workflows/diffpal-dev-review.yml)
[![npm](https://img.shields.io/npm/v/@diffpal/diffpal?label=npm)](https://www.npmjs.com/package/@diffpal/diffpal)
[![license: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Product Hunt](https://img.shields.io/badge/Product%20Hunt-DA552F?logo=producthunt&logoColor=white)](https://www.producthunt.com/products/diffpal)

**Open-source AI pull request review that runs in your CI.**

DiffPal is for teams that want AI review feedback without adopting a mandatory
hosted review service. Bring Codex, Copilot, OpenCode, another supported
provider, or any ACP-compatible agent, and keep one review workflow across
GitHub, GitLab, and Azure DevOps.

[Quickstart](docs/getting-started/github-quickstart.md) ·
[Documentation](docs/README.md) ·
[Live Demo](https://github.com/diffpal/demo/pull/13) ·
[Security](docs/security.md) ·
[GitHub](https://github.com/diffpal/diffpal)

## What Users Get

| Output | Where it shows up |
| --- | --- |
| PR/MR summary | GitHub reviews, GitLab summaries, Azure PR threads |
| Actionable findings | Inline comments, discussions, or PR threads on changed lines |
| Machine-readable artifacts | Findings JSON, summary Markdown, SARIF, and Code Quality reports |
| Optional merge gates | CI exit status, checks, commit statuses, or PR statuses |

## Why DiffPal

- **Runs in your CI:** review happens in the workflow you already control.
- **Provider choice:** use the default Codex copy-paste setup, another supported
  provider, or an ACP-compatible agent with the same DiffPal workflow.
- **Repository-owned configuration:** review policy, instructions, artifacts,
  and gates live with the codebase.
- **No mandatory hosted service:** DiffPal standardizes review output without
  requiring a hosted DiffPal review platform.

## Minimal GitHub Quickstart

Generate a GitHub Actions config with the default Codex API-key recipe:

```bash
npx -y @diffpal/diffpal@latest init --wizard --setup codex-api-key --platform github
```

Add `OPENAI_API_KEY` as a repository secret, then download the workflow:

`OPENAI_API_KEY` is a provider credential. DiffPal runs in your CI and sends
review input to the provider you configure, so keep this secret out of
untrusted fork PR jobs.

```bash
mkdir -p .github/workflows
curl -fsSL \
  https://raw.githubusercontent.com/diffpal/diffpal/main/examples/ci/github-actions/codex-api-key.yml \
  -o .github/workflows/diffpal.yml
```

Open a same-repository pull request. After the first successful run, expect a
`DiffPal Review Summary`, inline findings when actionable issues exist, and
the retained `diffpal-review` artifact containing
`.artifacts/diffpal/findings.json`.

For full setup details, provider alternatives, and fork PR guidance, use the
[GitHub quickstart](docs/getting-started/github-quickstart.md).

## Review Before Committing

Run the latest published CLI directly:

```bash
npx -y @diffpal/diffpal@latest --profile local review uncommitted
```

The command changes the provider task: the backend inspects the uncommitted
state in its workspace snapshot with its own tools. DiffPal does not construct a
second Git snapshot or pass changed files one by one. See the
[CLI reference](docs/reference/cli.md#diffpal-review-uncommitted).

## Coding-Agent Plugin

DiffPal includes an instruction-only plugin for Codex, Claude Code, Grok Build,
GitHub Copilot CLI, Cursor, OpenCode, and Agent Plugins 1.0.0 clients. Its three
skills configure separate local and CI profiles, run review with explicit
provider/publishing boundaries, and merge safe CI automation for GitHub,
GitLab, Azure, or custom runners.

See the [plugin guide](plugins/diffpal/README.md) for host installation and the
exact setup, review, and CI skill names.

## Supported Integrations

| Host | Native outputs | Guide |
| --- | --- | --- |
| GitHub Actions | PR review summary, file-level review comments, SARIF | [GitHub Actions](docs/integrations/github-actions.md) |
| GitLab CI | MR summary, discussions, Code Quality, SARIF, status | [GitLab CI](docs/integrations/gitlab-ci.md) |
| Azure Pipelines | PR summary thread, PR threads, PR status | [Azure Pipelines](docs/integrations/azure-pipelines.md) |
| Custom CI/CD | Artifact-only review, or publishing through a supported code host | [Custom CI/CD](docs/integrations/custom-ci.md) |

GitHub users can also install the
[DiffPal Review action](https://github.com/marketplace/actions/diffpal-review).
Azure users can install the
[DiffPal Review extension](https://marketplace.visualstudio.com/items?itemName=diffpal.diffpal).

## Documentation By Goal

| Goal | Start here |
| --- | --- |
| Run the first GitHub review | [GitHub quickstart](docs/getting-started/github-quickstart.md) |
| Confirm the first run worked | [Verify First Review](docs/getting-started/verify-first-review.md) |
| Improve a working setup | [Next Steps](docs/getting-started/next-steps.md) |
| Understand the concepts | [Concepts](docs/concepts/README.md) and [Glossary](docs/reference/glossary.md) |
| Choose a provider or ACP agent | [Providers](docs/providers/README.md) |
| Set up GitLab, Azure, or custom CI | [Integrations](docs/integrations/README.md) |
| Secure secrets and fork PRs | [Secrets and fork PRs](docs/guides/secrets-and-fork-prs.md) |
| Tune review policy | [Configuration reference](docs/reference/configuration.md) |
| Consume artifacts or schemas | [Artifacts](docs/reference/artifacts.md) and [findings schema](docs/reference/findings-schema.md) |
| Debug setup problems | [Troubleshooting](docs/help/troubleshooting.md) |
| Get quick answers | [FAQ](docs/help/faq.md) |
| Compare DiffPal to alternatives | [Comparison guide](docs/concepts/comparison.md) |

## Project Status

DiffPal is under active development. The public docs describe the current
supported workflow and integration surface.

## Contributing

Contributions are welcome. Start with [CONTRIBUTING.md](CONTRIBUTING.md) for
local setup, verification commands, and repository conventions.

## License

DiffPal is released under the [MIT License](LICENSE).
