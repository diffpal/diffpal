# CLI Reference

The public CLI entrypoint is `diffpal`.

## Installation

Verified distribution paths in this repository:

| Method | Command or entrypoint | Notes |
| --- | --- | --- |
| npm one-shot | `npx -y @diffpal/diffpal@latest <command>` | Used by the quickstart. |
| npm global | `npm install --global @diffpal/diffpal@latest` | Provides the `diffpal` executable. |
| GitHub Action | `diffpal/action@v1` | Wraps DiffPal for GitHub Actions. See [GitHub Actions](../integrations/github-actions.md). |
| Azure extension | `DiffPalReview@1` | Wraps DiffPal for Azure Pipelines. See [Azure Pipelines](../integrations/azure-pipelines.md). |

Release builds are produced by Omnidist for npm packages and platform binaries.
The npm meta-package is `@diffpal/diffpal`.

## Global Flags

| Flag | Default | Purpose |
| --- | --- | --- |
| `--config-dir` | empty | Extra config root directory searched before `.config/diffpal/config.yaml`. |
| `--debug` | `false` | Enable debug logging. |
| `--profile` | empty | Config profile name. |

## Command Overview

| Command | Purpose |
| --- | --- |
| `diffpal init` | Generate starter workspace configuration. |
| `diffpal review local` | Run provider-backed local review and print Markdown. |
| `diffpal review github` | Run review and publish/emit GitHub outputs. |
| `diffpal review gitlab` | Run review and publish/emit GitLab outputs. |
| `diffpal review ado` | Run review and publish/emit Azure DevOps outputs. Alias: `azure`. |
| `diffpal doctor` | Validate runtime, provider, workspace, and platform auth. |
| `diffpal debug prompt` | Render prompts and task snapshots without a provider call. |
| `diffpal sarif` | Convert a findings bundle to SARIF. |
| `diffpal version` | Print version metadata. |
| `diffpal completion` | Generate shell completion scripts. |

All public commands take flags only; there are no positional arguments in the
commands documented below.

## `diffpal init`

Generates `.config/diffpal/config.yaml`, `.config/diffpal/.gitignore`,
`.config/diffpal/templates/*`, `.config/diffpal/state/`, and `.diffpalignore`.

| Flag | Default | Allowed values |
| --- | --- | --- |
| `--wizard` | `false` | boolean |
| `--setup` | `codex-api-key` | `codex-api-key`, `codex-subscription`, `copilot-github-token`, `opencode-acp`, `generic-acp` |
| `--platform` | `auto` | `auto`, `github`, `gitlab`, `azure`, `none` |
| `--profile` | `ci` | letters, numbers, `.`, `_`, `-` |
| `--block-on` | `high` | `low`, `medium`, `high`, `critical` |
| `--config` | `.config/diffpal/config.yaml` | path |
| `--state` | `.config/diffpal/state` | path |
| `--force` | `false` | overwrite existing files |

Example:

```bash
npx -y @diffpal/diffpal@latest init --wizard --setup codex-api-key --platform github
```

## Review Commands

Shared review flags:

| Flag | Default | Purpose |
| --- | --- | --- |
| `--base` | empty | Base revision. Host commands can also resolve this from CI metadata. |
| `--head` | empty | Head revision. Host commands can also resolve this from CI metadata. |
| `--repo` | `local` | Repository ID used in deterministic fingerprints and links. |
| `--review-id` | command-specific | Review identifier used in deterministic output. |
| `--out` | `.artifacts/diffpal/findings.json` | Findings bundle output path. |
| `--language` | config value | Language for findings and summaries. |
| `--instructions` | config value | Replace config review instructions for this run. |
| `--instructions-file` | empty | Append instructions from a file. |
| `--block-on` | config value or `high` | Minimum blocking severity. |
| `--feedback` | `review` | `summary` or `review`. |
| `--gate` | `false` | Return non-zero when blocking findings exist. |

### `diffpal review local`

Outputs:

- writes the findings bundle to `--out`;
- prints Markdown summary or review output to stdout.

Example:

```bash
diffpal --profile ci review local \
  --base origin/main \
  --head HEAD \
  --feedback summary \
  --out .artifacts/diffpal/findings.json
```

### `diffpal review github`

Additional flags:

| Flag | Default | Purpose |
| --- | --- | --- |
| `--review-channel` | `diffpal` | GitHub publishing channel for PR reviews. |
| `--summary-overview` | `true` | Include semantic change overview in summaries. |
| `--dry-run` | `false` | Print host review Markdown without publishing. Supported for GitHub only. |

Environment inputs:

- `GITHUB_TOKEN` for publishing unless config auth supplies a token;
- `GITHUB_REPOSITORY`, `GITHUB_EVENT_PATH`, `GITHUB_EVENT_NAME` from GitHub
  Actions;
- `GITHUB_BASE_SHA` and `GITHUB_HEAD_SHA` as fallback base/head inputs;
- `DIFFPAL_GITHUB_API_URL` for tests or GitHub Enterprise API override.

Default surfaces:

- `summary`: SARIF and summary;
- `review`: PR review comments, SARIF, and summary.

Example:

```bash
diffpal --profile ci review github \
  --base "$BASE_SHA" \
  --head "$HEAD_SHA" \
  --repo "$GITHUB_REPOSITORY" \
  --review-id "github-pr-${PR_NUMBER}" \
  --feedback review \
  --gate
```

### `diffpal review gitlab`

`--dry-run` appears in the shared host help, but GitLab review rejects it.

Environment inputs:

- `GITLAB_TOKEN` or `CI_JOB_TOKEN` for publishing unless config auth supplies a
  token;
- `CI_PROJECT_PATH`, `CI_MERGE_REQUEST_IID`,
  `CI_MERGE_REQUEST_DIFF_BASE_SHA`, `CI_MERGE_REQUEST_SOURCE_BRANCH_SHA`,
  `CI_COMMIT_SHA`, `CI_COMMIT_SHORT_SHA`;
- `GITLAB_EVENT_PATH` or `CI_MERGE_REQUEST_EVENT_PATH` as payload fallback;
- `CI_JOB_URL` for status target URL;
- `DIFFPAL_GITLAB_API_URL` for tests or self-managed GitLab API override.

Default surfaces:

- `summary`: Code Quality, SARIF, status, and summary;
- `review`: Code Quality, discussions, status, SARIF, and summary.

### `diffpal review ado`

Alias: `diffpal review azure`.

`--dry-run` appears in the shared host help, but Azure review rejects it.

Environment inputs:

- `SYSTEM_ACCESSTOKEN` or `AZURE_DEVOPS_EXT_PAT` for publishing unless config
  auth supplies a token;
- `SYSTEM_COLLECTIONURI`, `SYSTEM_TEAMPROJECT`, `BUILD_REPOSITORY_NAME`,
  `BUILD_REPOSITORY_ID`, `BUILD_REPOSITORY_URI`, `BUILD_BUILDID`;
- `SYSTEM_PULLREQUEST_PULLREQUESTID`, `SYSTEM_PULLREQUEST_SOURCEBRANCH`,
  `SYSTEM_PULLREQUEST_TARGETBRANCH`, `SYSTEM_PULLREQUEST_SOURCECOMMITID`,
  `SYSTEM_PULLREQUEST_TARGETCOMMITID`;
- `BUILD_SOURCEVERSION` as a head/base fallback;
- `SYSTEM_PULLREQUEST_EVENT_PAYLOAD` as payload fallback.

Default surfaces:

- `summary`: status and summary;
- `review`: PR threads, status, and summary.

## `diffpal doctor`

Validates local runtime and environment.

| Flag | Default | Allowed values |
| --- | --- | --- |
| `--mode` | `local` | `local`, `github`, `gitlab`, `ado` |

Outputs diagnostic lines to stdout. Missing selected-provider auth is a warning
in local mode and an error in host modes. Platform auth is checked for host
modes.

```bash
diffpal --profile ci doctor --mode github
```

## `diffpal debug prompt`

Renders the system prompt, task snapshot, and a mock findings bundle without a
provider call. It still loads config and collects the diff.

Additional flags:

| Flag | Default | Allowed values |
| --- | --- | --- |
| `--format` | `text` | `text`, `json` |
| `--dry-run` | `true` | suppress normal review summary output |

It also accepts the shared review analysis and policy flags.

```bash
diffpal --profile ci debug prompt --base origin/main --head HEAD --format text
```

## `diffpal sarif`

Converts a findings bundle to SARIF.

| Flag | Default | Purpose |
| --- | --- | --- |
| `--input` | `.artifacts/diffpal/findings.json` | Input findings bundle. |
| `--out` | `.artifacts/diffpal/diffpal.sarif` | Output SARIF report. |

Output line:

```text
sarif=<path> findings=<count> version=<bundle-version>
```

## `diffpal version`

Prints:

```text
diffpal <version>+<git-commit> (<build-date>)
```

Development builds use `0.0.0-dev+unknown (unknown)`.

## `diffpal completion`

Generates Cobra shell completion scripts. Use `diffpal completion --help` for
the shell-specific subcommands generated by the CLI framework.
