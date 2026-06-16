# DiffPal

[![ci](https://github.com/diffpal/diffpal/actions/workflows/ci.yml/badge.svg)](https://github.com/diffpal/diffpal/actions/workflows/ci.yml)
[![diffpal-dev review](https://github.com/diffpal/diffpal/actions/workflows/diffpal-dev-review.yml/badge.svg)](https://github.com/diffpal/diffpal/actions/workflows/diffpal-dev-review.yml)
[![npm](https://img.shields.io/npm/v/@diffpal/diffpal?label=npm)](https://www.npmjs.com/package/@diffpal/diffpal)
[![license: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Diff-first AI review for pull requests.**

DiffPal turns changed code into structured findings, clear PR summaries, inline
comments, and merge gates across GitHub, GitLab, and Azure DevOps. Bring the
review agent your team already trusts: Codex, Copilot, OpenCode, Gemini, Claude
Code, a hosted provider, or any ACP-compatible CLI.

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

DiffPal owns the diff collection, finding schema, gating, and platform publish
logic. Your provider owns the model, tool loop, account, and credentials. That
split keeps CI setup predictable while still letting you choose the agent stack
your team already trusts.

Review instructions are produced by DiffPal's versioned Prompt Pack. Findings
artifacts include the prompt id, prompt version, purpose, and findings schema
version, so a review can be traced back to the exact prompt contract that
generated it. See the [config reference](docs/config-reference.md#prompt-pack)
and [findings schema](docs/findings-schema.md) for the current metadata.

## Quick Start: GitHub Actions

This is the fastest production-shaped setup: DiffPal installs itself through the
GitHub Action, Codex is used as the review provider, and `OPENAI_API_KEY` stays
in GitHub Secrets.

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

3. Add the workflow:

```bash
mkdir -p .github/workflows
cp examples/ci/github-actions/codex-api-key.yml .github/workflows/diffpal.yml
```

4. Open a same-repository pull request.

Expected result:

- a `diffpal-checks` check run
- a `DiffPal Review Summary` PR comment with an overview of the change
- inline comments only for actionable findings
- `.artifacts/diffpal/findings.json` in the job workspace
- a failed job only when `gate: true` and blocking findings exist, or when setup
  or publishing fails

The GitHub Action installs the latest DiffPal CLI by default. After your first
successful run, pin `diffpal-version`, provider CLIs, and bridge packages when
you need fully reproducible credentialed CI.

## Supported CI Systems

Use the same `.config/diffpal/config.yaml` shape in every CI system. The host
workflow only changes how it checks out code, installs the provider, passes the
platform token, and runs DiffPal.

| CI system | Examples | Output surfaces |
| --- | --- | --- |
| GitHub Actions | [`examples/ci/github-actions`](examples/ci/github-actions) | check run, PR summary, review comments, SARIF |
| GitLab CI | [`examples/ci/gitlab`](examples/ci/gitlab) | MR summary, discussions, Code Quality, SARIF |
| Azure Pipelines | [`examples/ci/azure-pipelines`](examples/ci/azure-pipelines) | PR summary thread, PR threads, PR status |

## Azure DevOps Marketplace Extension

Azure Pipelines users can install the public
[DiffPal Review extension](https://marketplace.visualstudio.com/items?itemName=diffpal.diffpal)
from the Azure DevOps Marketplace and add the `DiffPalReview@1` task to PR
validation pipelines.

The task installs `@diffpal/diffpal` by default, then runs `diffpal review ado`.
You still need a committed DiffPal config, a provider credential such as
`OPENAI_API_KEY`, `SYSTEM_ACCESSTOKEN` for PR feedback publishing, and a full git
checkout. See the [Azure Pipelines setup guide](docs/ci-examples.md#azure-pipelines)
for copy-paste examples.

GitHub Actions users can use the
[DiffPal Review action](https://github.com/marketplace/actions/diffpal-review):

```yaml
name: diffpal

on:
  pull_request:
    types: [opened, synchronize, reopened, ready_for_review]

jobs:
  review:
    name: review
    if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
    runs-on: ubuntu-latest
    permissions:
      contents: read
      pull-requests: write
      checks: write
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
    prompt_profile: v2
    strict_evidence: true
    strict_injection: true
    allow_nearby_context: true
    instructions: |
      Prefer actionable findings that are directly supported by the diff.
    checks:
      - security
      - bugs
      - performance
      - best-practices
  platforms:
    github: {}
    gitlab: {}
    azure: {}

profiles:
  ci:
    diffpal:
      gate:
        block_on: high
      review:
        prompt_profile: v2
        strict_evidence: true
        strict_injection: true
        allow_nearby_context: true
```

Review checks are intentionally simple. They ask the agent what to focus on;
DiffPal does not hardcode individual signal slugs:

| Check | Finding categories the agent may return |
| --- | --- |
| `security` | security |
| `bugs` | correctness, reliability |
| `performance` | performance |
| `best-practices` | maintainability, testing, style |

Severity is impact-based across all categories. The full critical/high/medium/low
matrix is in the [config reference](docs/config-reference.md#severity-matrix).

Use `diffpal.review.instructions`, the `instructions` action input, or
`--instructions-file` for repository-specific review guidance.

The review rollout fields are safe to canary per profile. Keep the repository
default conservative if needed, then set `profiles.ci.diffpal.review` to
`prompt_profile: v2`, `strict_evidence: true`, `strict_injection: true`, and
`allow_nearby_context: true` before making the gate blocking.

## Bring Your Own Agent

DiffPal is not a single-provider product. It delegates review to
`diffpal.provider`, which points at a provider under `runtime.providers`.

Ready-made config recipes. These are the same names accepted by
`diffpal init --wizard --setup ...`:

| Setup | Config | Secret |
| --- | --- | --- |
| Generic ACP CLI | [`examples/configs/generic-acp/config.yaml`](examples/configs/generic-acp/config.yaml) | provider-specific |
| Codex API key | [`examples/configs/codex-api-key/config.yaml`](examples/configs/codex-api-key/config.yaml) | `OPENAI_API_KEY` |
| Codex subscription auth | [`examples/configs/codex-subscription/config.yaml`](examples/configs/codex-subscription/config.yaml) | `CODEX_AUTH_JSON_B64` |
| Copilot fine-grained PAT | [`examples/configs/copilot-github-token/config.yaml`](examples/configs/copilot-github-token/config.yaml) | `COPILOT_GITHUB_TOKEN` |
| OpenCode ACP | [`examples/configs/opencode-acp/config.yaml`](examples/configs/opencode-acp/config.yaml) | OpenCode-specific |

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

Hosted providers receive DiffPal's read-only review tools during each review:
`git_changed_files`, `git_diff`, `list_files`, `read_file`, and
`search_files`. These are request-level tools, not provider config. ACP
providers keep their own tool surface.

For hosted providers, DiffPal records review tool usage in the findings bundle
and rejects a result when the provider did not inspect the diff with `git_diff`.
ACP providers use their native Git and filesystem tools, so DiffPal records that
runtime inspection proof is not available for that provider class.

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
| `summary` | One PR/MR summary plus check/status, no inline comments. |
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

That produces a separate `diffpal-dev-checks` check run and separate summary
comment.

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
