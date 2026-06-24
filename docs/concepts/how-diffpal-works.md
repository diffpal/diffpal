# How DiffPal Works

DiffPal is an open-source pull request and merge request review engine that
runs in your CI workflow. It standardizes how a repository asks an AI provider
for review, validates the response, publishes feedback, writes artifacts, and
optionally blocks a merge.

## Product Boundary

DiffPal owns the review workflow around the provider:

- collecting the requested `base..head` diff;
- building the review request;
- validating provider output against changed files and lines;
- rendering summaries, findings, and artifacts;
- publishing to supported code hosts;
- applying the configured gate.

DiffPal is not the model, the provider account, or a mandatory hosted review
service.

When a remote provider is configured, DiffPal sends review input to that
provider from the CI job. Use
[Security controls](../security.md) before enabling secret-backed review.

## Provider Boundary

The selected provider or ACP-compatible CLI owns model reasoning, model access,
provider-specific tools, credentials, sandboxing, and account management.
DiffPal chooses a provider through `diffpal.provider`, sends the review task,
and expects structured review output back.

Use [Providers and agents](providers-and-agents.md) for the provider model and
[Providers](../providers/README.md) for setup pages.

## Publisher Boundary

DiffPal publishers turn validated findings into host-native feedback. GitHub,
GitLab, and Azure DevOps have native publishers. Custom CI can still produce
local artifacts, and can publish through a supported code host when the job has
the required host metadata and credentials.

Use the [support matrix](../reference/support-matrix.md) for supported host
outputs.

## Repository-Owned Configuration

DiffPal reads `.config/diffpal/config.yaml` from the repository. That file owns
the selected provider, review settings, platform publishing settings, profiles,
and gate threshold. CI files install/authenticate the provider and pass host
context, but the review policy stays with the repository.

```mermaid
flowchart LR
    A[PR or MR event] --> B[CI job]
    B --> C[Repository DiffPal config]
    C --> D[DiffPal review engine]
    D --> E[Selected provider or ACP agent]
    E --> F[Structured findings]
    F --> G[Publisher]
    F --> H[Artifacts]
    F --> I[Optional gate]
```

For the first setup path, start with the
[GitHub quickstart](../getting-started/github-quickstart.md). For supported
hosts and outputs, see the [support matrix](../reference/support-matrix.md).
