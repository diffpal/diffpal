# GitHub Quickstart

Use this page to add DiffPal to a new GitHub repository and see the first PR
review.

This path uses the Codex API-key recipe because it is a complete copy-paste
GitHub Actions setup. Codex is not the product boundary; other supported
providers and ACP agents can use the same DiffPal workflow.
See [Codex Provider](../providers/codex.md) for provider-specific setup
details.

## 1. Create A Setup Branch

Run this from the repository root:

```bash
git switch -c diffpal-onboarding
```

## 2. Add The Provider Secret

Add this repository secret in GitHub:

| Secret | Purpose |
| --- | --- |
| `OPENAI_API_KEY` | Lets the Codex CLI act as the review provider. |

DiffPal runs in your CI and sends review input to the provider you configure.
For this setup, `OPENAI_API_KEY` belongs to the Codex provider, not to GitHub.
Store it as a repository secret and do not expose it to untrusted fork PR code.

With GitHub CLI:

```bash
gh secret set OPENAI_API_KEY
```

Keep secret-backed DiffPal review limited to same-repository pull requests and
let forks run no-secret CI only. See
[Secrets and fork PRs](../guides/secrets-and-fork-prs.md) for the security
rationale.

## 3. Initialize Config

Run this from the repository root:

```bash
npx -y @diffpal/diffpal@latest init --wizard --setup codex-api-key --platform github
```

This creates `.config/diffpal/config.yaml` with Codex ACP as the provider,
`block_on: high`, a `ci` profile, and a GitHub platform block. Existing files
are preserved unless you pass `--force`.

Commit the generated config:

```bash
git add .config/diffpal/config.yaml .config/diffpal/.gitignore
git commit -m "chore: add diffpal config"
```

## 4. Install Workflow

Copy the GitHub Actions example:

```bash
mkdir -p .github/workflows
cp examples/ci/github-actions/codex-api-key.yml .github/workflows/diffpal.yml
```

Commit the workflow:

```bash
git add .github/workflows/diffpal.yml
git commit -m "ci: add diffpal review"
```

The workflow performs a full checkout, installs the Codex provider command,
authenticates with `OPENAI_API_KEY`, runs `diffpal/action@v1`, publishes review
feedback, and enables the gate.

## 5. Test A Trusted Pull Request

Push the branch and open a pull request from a branch in the same repository:

```bash
git push -u origin HEAD
```

Use a same-repository pull request for the first test so the provider secret is
available to the workflow.

## Expected Result

After the workflow completes, the pull request should show:

- a `DiffPal Review Summary` review;
- inline review comments when actionable findings exist;
- a `diffpal` workflow check;
- `.artifacts/diffpal/findings.json` in the workflow workspace.

If the run has no actionable findings, the review summary and artifacts should
still appear. If setup, authentication, diff collection, or publishing fails,
the workflow should fail because the review is incomplete.

Use [Verify First Review](verify-first-review.md) to check the first run, then
continue with [Next Steps](next-steps.md).
