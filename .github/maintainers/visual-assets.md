# Maintainer Visual Assets

This page is a maintainer checklist for future repository visuals. It is not
part of the public user documentation hierarchy.

## Hero Composite Screenshot

Show one pull request viewport containing:

- a `DiffPal Review Summary`
- one actionable inline finding
- the DiffPal check or status
- a small callout for `.artifacts/diffpal/findings.json`

## First-Run GIF

Show the shortest successful path:

- run `diffpal init --wizard`
- commit `.config/diffpal/config.yaml`
- copy the CI workflow
- open a same-repository pull request
- see summary, inline finding, and findings bundle

## Architecture Diagram

The README uses this Mermaid diagram until a designed diagram exists:

```mermaid
flowchart LR
    A[Pull request diff] --> B[DiffPal config and policy]
    B --> C[Selected provider or ACP CLI]
    C --> D[Structured findings bundle]
    D --> E[PR summary and inline feedback]
    D --> F[SARIF and Code Quality artifacts]
    D --> G[Merge gates and statuses]
```

## Feedback Modes Visual

Create a side-by-side image for:

- `summary`: summary plus non-file artifacts
- `review`: summary plus platform-native file-level findings
