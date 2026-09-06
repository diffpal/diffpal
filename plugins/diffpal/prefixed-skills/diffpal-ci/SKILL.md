---
name: diffpal-ci
description: Configure DiffPal review in CI safely. Use when adding or updating DiffPal in GitHub Actions, GitLab CI, Azure Pipelines, or a custom CI system; selecting a CI provider/profile; configuring artifacts, feedback, or gates; or protecting review credentials from forked contributions.
---

# Configure DiffPal CI

Add DiffPal to the repository's existing CI structure without exposing credentials or discarding unrelated jobs.

## Workflow

1. Read repository instructions, .config/diffpal/config.yaml, existing workflow files, dependency/version policy, and the checked-in DiffPal integration examples when available.
2. Detect the CI platform. If it is ambiguous, ask which existing workflow or job should receive DiffPal.
3. Read exactly one matching reference:
   - [GitHub Actions](../../references/ci/github-actions.md)
   - [GitLab CI](../../references/ci/gitlab-ci.md)
   - [Azure Pipelines](../../references/ci/azure-pipelines.md)
   - [Custom CI](../../references/ci/custom-ci.md)

   Do not load the other platform references unless the user asks for a comparison or multiple platforms.
4. Resolve these choices before drafting:
   - ci profile and its configured provider;
   - provider installation/authentication recipe and deliberately selected versions;
   - artifact-only versus native host publishing;
   - summary versus review feedback;
   - gate off versus --gate, including the blocking threshold;
   - target workflow file and job.
5. Parse the existing workflow and design a minimal merge. Preserve unrelated triggers, permissions, jobs, steps, includes, templates, variables, comments where practical, and repository version-pinning conventions.
6. Audit the proposal for complete history, base/head/repository/review identity, provider setup, separate provider and host credential references, trusted-source protection, .artifacts/diffpal/ upload, feedback, and gate behavior.
7. Preview the exact workflow diff, chosen versions, secret names, permissions, publication surfaces, and fork behavior. Never show secret values.
8. Obtain explicit approval before writing the workflow. Apply only the approved merge, then parse it and run available local static checks.
9. Report validation results and manual steps. Do not set secrets, install tools, push, dispatch a workflow, approve a protected environment, or publish a review without separate authorization.

## Safety rules

- Use --profile ci or the native integration's profile: ci.
- Provider credentials authorize repository-content processing; host credentials authorize publication. Keep them separate and expose neither to untrusted fork code.
- Artifact-only review still calls a provider and therefore still requires trusted handling of provider credentials.
- Prefer same-repository/trusted-source guards and maintainer-controlled reruns over secret-bearing fork jobs.
- Treat feedback and gating as independent choices. review controls detail; --gate controls the process result.
- Never replace an existing workflow wholesale when a structural merge is possible.
