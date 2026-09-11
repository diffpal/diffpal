# GitHub Actions

Merge DiffPal into an existing file under `.github/workflows/`; create a focused workflow only when no suitable file exists.

## Shared contract

- Use `pull_request`, never `pull_request_target` to execute contributor code.
- Check out full history with `actions/checkout@v4` and `fetch-depth: 0`.
- Pass base from `github.event.pull_request.base.sha`, head from `github.event.pull_request.head.sha`, repository from `github.repository`, and a stable review ID derived from `github.event.pull_request.number`.
- Install the selected DiffPal CLI and configured provider with deliberate versions that follow repository pinning policy.
- Run `<diffpal-command> --profile ci doctor --mode local` with the same executable before review.
- Upload `.artifacts/diffpal/` with `actions/upload-artifact@v4` under `if: always()`, without masking the saved review exit status.

## Trust and configuration

A same-repository condition such as the following rejects forks, but does not establish source trust:

```yaml
if: ${{ !github.event.pull_request.draft && github.event.pull_request.head.repo.full_name == github.repository }}
```

The checked-out PR can alter `.config/diffpal/config.yaml`, including a provider `cmd`. Before a secret-bearing provider step, use an organization-approved actor/branch policy plus a protected environment or maintainer approval, or load a trusted config from the base branch or an external location. For the latter, place `diffpal/config.yaml` or `config.yaml` under a trusted root, pass `--config-dir <trusted-config-root>` to both doctor and review, and verify that the expected file exists so lookup cannot fall back to PR-controlled workspace config. Do not execute PR-controlled install scripts or commands before this trust boundary.

## Artifact-only mode

- Keep job permissions at `contents: read`; do not grant `pull-requests: write`.
- Do not pass `GITHUB_TOKEN` to DiffPal. The job still needs the provider credential after its trust guard.
- Invoke `review local` with explicit base, head, repo, review ID, feedback, output, and optional gate:

```bash
<diffpal-command> --profile ci review local \
  --base "${{ github.event.pull_request.base.sha }}" \
  --head "${{ github.event.pull_request.head.sha }}" \
  --repo "${{ github.repository }}" \
  --review-id "github-pr-${{ github.event.pull_request.number }}" \
  --feedback <summary-or-review> \
  --out .artifacts/diffpal/findings.json
```

Append `--gate` when gating is enabled.

Capture the Markdown stdout as a job summary or artifact without turning it into GitHub review comments.

## Native publishing mode

- Grant `pull-requests: write` only to the publishing job and pass GitHub's `GITHUB_TOKEN` separately from the provider credential.
- Use `diffpal/action@v1` with explicit `profile: ci`, base, head, repo, review ID, feedback, and gate, or invoke `review github` with the equivalent explicit values.
- If the action cannot receive the required trusted `--config-dir`, use the CLI path instead.
- Explain whether `feedback: summary` or `feedback: review` will publish file-level feedback and whether the gate can fail a required check.

## Merge and handoff

Preserve existing events, concurrency, jobs, matrices, permissions, and pinning conventions. Preview the trust predicate, environment approval, config origin, permissions, secret names, publication surface, and fork behavior. Do not configure secrets, push, dispatch, approve an environment, or publish during setup.
