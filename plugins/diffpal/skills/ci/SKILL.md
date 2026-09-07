---
name: ci
description: Configure DiffPal review in CI safely. Use when adding or updating DiffPal in GitHub Actions, GitLab CI, Azure Pipelines, or a custom CI system; selecting a CI provider/profile; configuring artifacts, feedback, or gates; or protecting review credentials and trusted configuration from pull-request changes.
---

# Configure DiffPal CI

Merge DiffPal into the repository's existing CI structure without exposing credentials, trusting contributor-controlled commands, or discarding unrelated jobs.

## Inspect and validate

1. Read repository instructions, `.config/diffpal/config.yaml`, existing workflow files, dependency/version policy, and checked-in DiffPal integration examples.
2. Detect the CI platform and target workflow or job. If that choice is materially ambiguous, ask before editing.
3. Read exactly one matching reference:
   - [GitHub Actions](../../references/ci/github-actions.md)
   - [GitLab CI](../../references/ci/gitlab-ci.md)
   - [Azure Pipelines](../../references/ci/azure-pipelines.md)
   - [Custom CI](../../references/ci/custom-ci.md)

   Load another platform reference only for a requested comparison or multi-platform change.
4. Select one exact DiffPal executable and version using repository policy. Before drafting the workflow, validate the effective CI configuration with that same executable:

```bash
<diffpal-command> --profile ci doctor --mode local
```

   This checks configuration and local executables, not provider authentication or connectivity. If the `ci` profile is missing or invalid, route configuration repair through the setup skill or include it as a separately previewed and approved change.

## Resolve the contract

5. Resolve all of these choices:
   - artifact-only `review local` versus native `review github`, `review gitlab`, or `review ado` publishing;
   - effective `ci` provider, provider install/auth recipe, and deliberate versions;
   - summary versus review feedback;
   - gate off versus `--gate`, including its blocking threshold;
   - base, head, repository ID, stable review ID, artifact path, and retention;
   - the exact trusted-source predicate and the origin of the runtime DiffPal config.
6. Treat same-repository or non-fork checks only as transport filters. They do not make a source checkout trusted: contributor-controlled config can change a provider `cmd`. Before exposing a provider credential, require one of:
   - an organization-approved trusted actor or branch plus a maintainer/protected-environment approval; or
   - DiffPal config loaded from a trusted base ref or external location via `--config-dir`, with no contributor-controlled command executed before the trust boundary.
7. Keep provider credentials separate from host-publishing credentials. Artifact-only mode needs provider authentication but no host write token or host publication permission.

## Propose, apply, and verify

8. Design a minimal structural merge. Preserve unrelated triggers, permissions, jobs, steps, includes, templates, variables, comments, and version-pinning conventions.
9. Preview the exact workflow and config diffs, selected versions, secret names, permissions, trust predicate, config origin, publication surfaces, feedback, gate behavior, and fork behavior. Never show secret values.
10. Obtain explicit approval before repository writes. Apply only the approved merge, then parse the changed workflow and run available local static checks.
11. Report validation results and manual administrator steps. Do not set secrets, install tools, push, dispatch a workflow, approve a protected environment, or publish a review without separate authorization.

## Safety rules

- Use `--profile ci` or the native integration's `profile: ci`.
- Upload `.artifacts/diffpal/` even when review or gating fails, while preserving the DiffPal exit status.
- Treat feedback and gating independently: feedback controls output detail; gating controls process failure.
- Never use `pull_request_target` to execute contributor code with secrets.
- Never replace an existing workflow wholesale when a structural merge is possible.
