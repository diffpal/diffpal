# DiffPal

[![ci](https://github.com/diffpal/diffpal/actions/workflows/ci.yml/badge.svg)](https://github.com/diffpal/diffpal/actions/workflows/ci.yml)
[![diffpal-dev review](https://github.com/diffpal/diffpal/actions/workflows/diffpal-dev-review.yml/badge.svg)](https://github.com/diffpal/diffpal/actions/workflows/diffpal-dev-review.yml)
[![npm](https://img.shields.io/npm/v/@diffpal/diffpal?label=npm)](https://www.npmjs.com/package/@diffpal/diffpal)
[![license: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Open-source, provider-agnostic AI review for pull requests.**

DiffPal is an open-source PR review system that turns changed code into
structured findings, clear summaries, inline feedback, artifacts, and merge
gates. Teams bring their own AI provider or ACP-compatible CLI, so there is no
mandatory hosted DiffPal review service and no required per-seat review
platform.

DiffPal exists to make AI code review something teams control, not another
review platform they rent. It runs in your CI, uses the AI provider you choose,
and turns every pull request into clear summaries, actionable inline feedback,
review artifacts, and merge gates.

Use the provider path that already works for your team and keep the review
workflow in your repository. DiffPal's goal is to make AI PR review portable,
affordable, and enforceable across GitHub, GitLab, and Azure DevOps.

| Works with | Publishes | Gates on |
| --- | --- | --- |
| GitHub Actions, GitLab CI, Azure Pipelines | summaries, inline comments, checks/statuses, SARIF, Code Quality | policy thresholds through native CI checks and PR statuses |

[Quickstart](docs/quickstart.md) · [CI examples](docs/ci-examples.md) · [Config reference](docs/config-reference.md) · [Provider recipes](examples/README.md)

## How DiffPal Works

```text
pull request diff
  -> DiffPal config and policy
  -> selected ACP or hosted provider
  -> structured findings bundle
  -> platform publisher and CI artifacts
```

DiffPal coordinates the review workflow around the model call. Your provider
supplies the review intelligence; DiffPal keeps PR feedback, artifacts, and
merge policy consistent across hosts.

Review instructions are produced by DiffPal's versioned Prompt Pack. Findings
artifacts include the prompt id, prompt version, purpose, and findings schema
version, so a review can be traced back to the exact prompt contract that
generated it. See the [config reference](docs/config-reference.md#prompt-pack)
and [findings schema](docs/findings-schema.md) for the current metadata.

## Bring Your Own Provider

DiffPal decouples AI review from any one vendor or hosted service. Choose
Codex, Copilot, OpenCode, Gemini, Claude Code, a hosted API provider, an ordered
provider pool, or any ACP-compatible CLI without rebuilding your PR review
workflow.

That model keeps cost control with your team. DiffPal does not require a hosted
review service or per-seat platform subscription to collect diffs, publish PR
feedback, write artifacts, or enforce merge gates.

## Quick Start: GitHub Actions

This is the fastest production-shaped setup using the default Codex API-key
recipe: DiffPal installs itself through the GitHub Action, Codex is selected as
the review provider, and `OPENAI_API_KEY` stays in GitHub Secrets. You can swap
the provider recipe while keeping the same DiffPal review workflow.

1. Generate the config:

```bash
npx -y @diffpal/diffpal@latest init --wizard --setup codex-api-key --platform github
```

This writes `.config/diffpal/config.yaml` with a visible `ci` profile. Existing
files are kept unless you pass `--force`.

2. Add a repository secret:

| Secret | Purpose |
| --- | --- |
| `OPENAI_API_KEY` | Authenticates Codex for the review provider. |

For public open-source repositories, keep provider credentials away from fork PR
code. GitHub's fork workflow approval settings control whether outside
contributors' fork workflows run automatically; they do not make it safe to
release provider secrets to fork code. Keep DiffPal's secret-backed review job
limited to same-repository pull requests. Fork PRs should run no-secret CI only.

3. Add the workflow:

```bash
mkdir -p .github/workflows
cp examples/ci/github-actions/codex-api-key.yml .github/workflows/diffpal.yml
```

4. Open a same-repository pull request.

Expected result:

- a `DiffPal Review Summary` PR review with an overview of the change
- inline review comments for actionable findings
- `.artifacts/diffpal/findings.json` in the job workspace
- a failed job only when `gate: true` and blocking findings exist, or when setup
  or publishing fails

The GitHub Action installs the latest DiffPal CLI by default. After your first
successful run, pin `diffpal-version`, provider CLIs, and bridge packages when
you need fully reproducible credentialed CI.

## Supported CI Systems

Use the same `.config/diffpal/config.yaml` shape in every CI system. GitHub,
GitLab, and Azure are publishing targets; the core workflow only changes how CI
checks out code, installs the provider, passes the platform token, and runs
DiffPal.

| CI system | Examples | Output surfaces |
| --- | --- | --- |
| GitHub Actions | [`examples/ci/github-actions`](examples/ci/github-actions) | PR review summary, inline review comments, SARIF |
| GitLab CI | [`examples/ci/gitlab`](examples/ci/gitlab) | MR summary, discussions, Code Quality, SARIF |
| Azure Pipelines | [`examples/ci/azure-pipelines`](examples/ci/azure-pipelines) | PR summary thread, PR threads, PR status |

## GitHub Action

GitHub Actions users can install the
[DiffPal Review action](https://github.com/marketplace/actions/diffpal-review)
with `uses: diffpal/action@v1`. The action source and release automation live in
the separate [diffpal/action](https://github.com/diffpal/action) repository.

The action installs `@diffpal/diffpal` by default, then runs
`diffpal review github`. You still own provider setup and authentication in the
workflow, so switching provider recipes does not require switching PR review
platforms.

## Azure DevOps Marketplace Extension

Azure Pipelines users can install the public
[DiffPal Review extension](https://marketplace.visualstudio.com/items?itemName=diffpal.diffpal)
from the Azure DevOps Marketplace and add the `DiffPalReview@1` task to PR
validation pipelines. Extension source and release automation live in the
separate [diffpal/azure-devops](https://github.com/diffpal/azure-devops)
repository.

The task installs `@diffpal/diffpal` by default, then runs `diffpal review ado`.
You still need a committed DiffPal config, a provider credential such as
`OPENAI_API_KEY`, `SYSTEM_ACCESSTOKEN` for PR feedback publishing, and a full git
checkout. See the [Azure Pipelines setup guide](docs/ci-examples.md#azure-pipelines)
for copy-paste examples.

Example GitHub Actions workflow:

```yaml
name: diffpal

on:
  pull_request:
    types: [opened, synchronize, reopened, ready_for_review]

jobs:
  review:
    name: review
    # Provider credentials are only exposed to same-repository PRs.
    # Fork PRs should run no-secret CI only.
    if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
    steps:
      - uses: actions/checkout@v6
        with:
          fetch-depth: 0

      - uses: actions/setup-node@v6
        with:
          node-version: 22

      - name: Install Codex provider
        run: npm install --global @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3

      - name: Authenticate Codex
        run: printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
        env:
          OPENAI_API_KEY: ${{ secrets.OPENAI_API_KEY }}

      - uses: diffpal/action@v1
        with:
          base: ${{ github.event.pull_request.base.sha }}
          head: ${{ github.event.pull_request.head.sha }}
          profile: ci
          gate: true
          feedback: balanced
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

`pull_request_target` runs from the default branch of the base repository and is
useful for trusted automation such as labeling or commenting. Do not combine it
with checking out the PR head or running package installs, tests, build scripts,
hooks, provider CLIs, or other fork code. That pattern can expose privileged
tokens or secrets to untrusted code.

If you prefer copying files manually, use
[`examples/configs/codex-api-key/config.yaml`](examples/configs/codex-api-key/config.yaml).
For full copy-paste files and host-specific notes, read
[`docs/ci-examples.md`](docs/ci-examples.md).

## Config You Commit

DiffPal uses the current profile-based config shape. There is no legacy
`defaults` block.

```yaml
version: v1

runtime:
  providers:
    codex-acp:
      type: codex_acp
      codex_acp:
        reasoning_effort: low

diffpal:
  provider: codex-acp
  gate:
    block_on: high
  review:
    language: en
    instructions: |
      Prefer actionable findings that are directly supported by the diff.
  platforms:
    github: {}
    gitlab: {}
    azure: {}

profiles:
  ci:
    diffpal:
      gate:
        block_on: high
```

DiffPal uses a fixed finding taxonomy: security, correctness, reliability,
performance, maintainability, testing, and style. Use review instructions to
change or extend the review scope, for example `Review for OWASP best practices
and authz/authn regressions.`

Severity is impact-based across all categories. The full critical/high/medium/low
matrix is in the [config reference](docs/config-reference.md#severity-matrix).

Use `diffpal.review.instructions`, the `instructions` action input, or
`--instructions-file` for repository-specific review guidance.

## Provider Recipes and Runtime Types

DiffPal delegates review to `diffpal.provider`, which points at a provider
under `runtime.providers`.

Ready-made config recipes. These are the same names accepted by
`diffpal init --wizard --setup ...`:

| Setup | Config | Secret |
| --- | --- | --- |
| Generic ACP CLI | [`examples/configs/generic-acp/config.yaml`](examples/configs/generic-acp/config.yaml) | provider-specific |
| Codex API key | [`examples/configs/codex-api-key/config.yaml`](examples/configs/codex-api-key/config.yaml) | `OPENAI_API_KEY` |
| Codex subscription auth | [`examples/configs/codex-subscription/config.yaml`](examples/configs/codex-subscription/config.yaml) | `CODEX_AUTH_JSON_B64` |
| Copilot fine-grained PAT | [`examples/configs/copilot-github-token/config.yaml`](examples/configs/copilot-github-token/config.yaml) | `COPILOT_GITHUB_TOKEN` |
| OpenCode ACP | [`examples/configs/opencode-acp/config.yaml`](examples/configs/opencode-acp/config.yaml) | OpenCode-specific |

For Codex subscription auth, generate a fresh `CODEX_AUTH_JSON_B64` value with
the command recipe in [`examples/README.md`](examples/README.md#generate-codex_auth_json_b64).

Use `generic_acp` for any CLI that can start an ACP stdio server:

```yaml
runtime:
  providers:
    my-review-agent:
      type: generic_acp
      generic_acp:
        cmd: ["your-acp-cli", "acp", "--stdio"]

diffpal:
  provider: my-review-agent
```

Common provider aliases are available when the CLI is already installed and
authenticated:

| Type | Runtime command |
| --- | --- |
| `codex_acp` | Codex ACP via the configured Codex bridge |
| `copilot_acp` | Copilot ACP |
| `opencode_acp` | `opencode acp` |
| `gemini_acp` | Gemini ACP |
| `claude_code_acp` | Claude Code ACP |
| `generic_acp` | Your explicit command |
| `openai`, `aistudio` | Hosted API providers |
| `pool` | Ordered provider failover |

DiffPal passes the review task snapshot with base and head revisions. Providers
inspect the repository diff and supporting code through their available Git and
filesystem tools, then DiffPal validates the structured findings against the
changed ranges it collected internally.

## MCP Servers

DiffPal can pass MCP servers through the Norma runtime to providers that support
them. Declare servers once, then attach them to the provider:

```yaml
runtime:
  mcp_servers:
    repo-docs:
      type: stdio
      cmd: ["your-docs-mcp-server"]
      args: ["--root", "."]
      env:
        DOCS_TOKEN: "${DOCS_TOKEN}"
    policy-api:
      type: http
      url: "https://policy.example.com/mcp"
      headers:
        Authorization: "Bearer ${POLICY_MCP_TOKEN}"
  providers:
    opencode-acp:
      type: opencode_acp
      mcp_servers:
        - repo-docs
        - policy-api
      opencode_acp:
        model: opencode/big-pickle

diffpal:
  provider: opencode-acp
```

Keep MCP credentials in CI secrets. Use envsubst placeholders only for values
that are guaranteed to exist in that job.

## Feedback Modes

Use `feedback` for the normal user-facing shape:

| Mode | Behavior |
| --- | --- |
| `summary` | One PR/MR summary. On GitHub, DiffPal still publishes actionable findings as inline PR review comments. |
| `balanced` | Summary plus actionable high-confidence inline feedback. |
| `inline` | Summary plus a more permissive inline threshold. |

Use `summary-overview: false` or `--summary-overview=false` if you do not want
the semantic change overview in the summary.

If two DiffPal workflows run on the same PR, give them separate channels:

```yaml
with:
  review-channel: diffpal-dev
  review-id: github-pr-${{ github.event.pull_request.number }}-diffpal-dev
```

That produces a separate `diffpal-dev` PR review with its own summary and inline
comments.

## Local Debugging

CI is the main path, but local checks help when wiring a provider:

```bash
npm install --global @diffpal/diffpal@latest @openai/codex@0.139.0 @normahq/codex-acp-bridge@1.6.3
printf '%s' "$OPENAI_API_KEY" | codex login --with-api-key
diffpal doctor --mode github
diffpal review local --base origin/main --head HEAD --profile ci
```

To inspect the prompt contract without calling any provider:

```bash
diffpal debug prompt --base origin/main --head HEAD --profile ci --format text
```

The debug command renders the system prompt, the review task snapshot, and a
schema-valid mock findings bundle through the normal review validation path.
It does not require API keys.

## Documentation

- [Quickstart](docs/quickstart.md)
- [CI setup guide](docs/ci-examples.md)
- [Examples](examples/README.md)
- [Config reference](docs/config-reference.md)
- [Findings schema](docs/findings-schema.md)
- [GitLab adapter reference](docs/platform-gitlab.md)
- [Azure adapter reference](docs/platform-azure.md)
- [Release process](docs/release.md)
- [Contributing](CONTRIBUTING.md)
