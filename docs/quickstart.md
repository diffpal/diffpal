# DiffPal Quickstart

This guide gets DiffPal running with the Copilot ACP provider and one CI
system. For production workflows, pin package versions after the first working
setup.

## 1. Install Locally

```bash
npm install --global @diffpal/diffpal@latest @github/copilot@latest
diffpal version
diffpal doctor
```

`diffpal` is the review CLI. `copilot` is the default AI provider command used
by the generated config.

## 2. Create Config

Run this at the root of your repository:

```bash
diffpal init
```

This writes:

- `.config/diffpal/config.yaml`
- `.config/diffpal/templates/`
- `.diffpalignore`

The generated config uses `copilot-acp` by default and reviews for:

- `bugs`
- `performance`
- `best-practices`

Edit `.config/diffpal/config.yaml` if you want a different language,
threshold, context size, or provider.

## 3. Add Secrets

DiffPal needs two kinds of credentials in CI:

| Secret | Purpose |
| --- | --- |
| `COPILOT_GITHUB_TOKEN` | Lets the Copilot CLI act as the review provider. |
| Platform token | Lets DiffPal publish comments/checks/statuses back to the PR. |

Platform tokens:

- GitHub Actions: built-in `GITHUB_TOKEN`
- GitLab CI: built-in `CI_JOB_TOKEN` or a `GITLAB_TOKEN`
- Azure Pipelines: built-in `SYSTEM_ACCESSTOKEN`

Do not commit token values into `.config/diffpal/config.yaml`. Use CI secrets or
the standard environment variables above.

## 4. Check the Environment

Before enabling a blocking gate, run the relevant doctor check:

```bash
diffpal doctor --mode github
diffpal doctor --mode gitlab
diffpal doctor --mode ado
```

Doctor validates config, provider selection, and platform auth for the selected
mode.

## 5. Run a Local Review

```bash
diffpal review local \
  --base origin/main \
  --head HEAD \
  --out .artifacts/diffpal/findings.json
```

Local review writes a structured findings bundle and does not publish comments.

## 6. Add CI

Use the full copy-paste setup guide for your platform:

- [GitHub Actions](ci-examples.md#github-actions)
- [GitLab CI](ci-examples.md#gitlab-ci)
- [Azure Pipelines](ci-examples.md#azure-pipelines)

Each recipe includes required permissions, secrets, install steps, and expected
PR output.

## Common Review Controls

These flags work with `review local`, `review github`, `review gitlab`, and
`review ado`:

```bash
--language en
--review-checks bugs,performance,best-practices
--block-on high
--gate
```

- `--language`: language for generated review text.
- `--review-checks`: checks to run.
- `--block-on`: severity threshold that marks findings as blocking.
- `--gate`: exits non-zero when blocking findings exist.

## What Success Looks Like

After a CI run, expect:

- a DiffPal summary comment or status/check
- inline comments only when actionable findings exist
- `.artifacts/diffpal/findings.json`
- exit code `0` when the review passes

If setup fails, start with:

```bash
diffpal doctor --mode <github|gitlab|ado>
```
