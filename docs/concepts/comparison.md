# Comparison Guide

DiffPal is best understood as an open-source control plane for AI pull request
review. It does not try to be the model, the hosted review platform, or a
general lint result router.

## DiffPal Vs Hosted AI Reviewers

Use DiffPal when you want:

- provider choice through `runtime.providers`
- review config, instructions, artifacts, and gate policy committed with the
  repository
- CI-native execution without a mandatory hosted DiffPal review service
- consistent outputs across GitHub, GitLab, and Azure DevOps

Hosted reviewers can be a better fit when your team wants a fully managed
product experience, centralized account management, or provider decisions made
for you.

## DiffPal Vs Lint Publishers

Lint publishers take existing tool output and annotate pull requests. DiffPal
resolves the review scope, asks a selected AI provider for review findings,
validates the structured output, and publishes both human feedback and
machine-readable artifacts.

Use both together when you want deterministic linters plus policy-aware AI
review in the same CI workflow.

## DiffPal Vs CI Convention Bots

CI convention bots are good at enforcing explicit repository rules. DiffPal is
for review feedback that needs model reasoning over a pull request diff while
still producing auditable findings and merge gates.

Use DiffPal when the important choice is not "which hosted reviewer do we buy?"
but "how do we standardize AI review while keeping control of provider,
workflow, and artifacts?"
