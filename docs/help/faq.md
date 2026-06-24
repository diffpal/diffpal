# FAQ

## Is A Hosted DiffPal Service Required?

No. DiffPal runs in your CI. It can publish to supported code hosts from that
CI job when you provide host credentials.

See [How DiffPal works](../concepts/how-diffpal-works.md).

## Can Existing Agents Be Used?

Yes, when they are exposed through a supported provider type or an
ACP-compatible CLI. DiffPal selects the provider from `diffpal.provider` and
`runtime.providers`.

See [Providers](../providers/README.md) and
[Providers and agents](../concepts/providers-and-agents.md).

## Which Hosts Are Supported?

DiffPal has native publishers for GitHub pull requests, GitLab merge requests,
and Azure DevOps pull requests. Unsupported code hosts can still use local
Markdown and artifacts through custom CI.

See the [support matrix](../reference/support-matrix.md).

## Is Custom CI Supported?

Yes. Jenkins, Buildkite, CircleCI, Bitbucket Pipelines, internal runners, and
other CI systems can run DiffPal through the stable custom CI contract. That
does not imply native support for those CI products.

See [Custom CI/CD](../integrations/custom-ci.md).

## How Are Provider Costs And Accounts Handled?

DiffPal does not own, create, bill, or manage third-party provider accounts.
Provider credentials, quotas, billing, and model access belong to the provider
account you configure in CI.

See [Providers](../providers/README.md) and
[Secrets and fork PRs](../guides/secrets-and-fork-prs.md).

## What Happens When No Findings Exist?

A completed review can legitimately produce no findings. The summary and
artifacts should still show that the review ran. With no blocking findings, an
enabled gate passes.

See [Verify First Review](../getting-started/verify-first-review.md) and
[Exit behavior](../reference/exit-behavior.md).

## Can Comments And Gates Be Configured Independently?

Yes. `feedback` controls summary versus inline feedback. `--gate` or
`gate: true` controls whether blocking findings fail the job. A run can publish
feedback without blocking merges, or block based on findings after output is
written.

See [Findings, feedback, and gates](../concepts/findings-feedback-and-gates.md)
and [Merge gates](../integrations/README.md#merge-gates).

## How Are Fork PRs Handled?

Secret-backed DiffPal review should run only for trusted branches,
same-repository pull requests, or maintainer-approved jobs that do not execute
fork-controlled code with secrets. Fork PRs should run no-secret CI unless a
maintainer uses a safe trusted workflow.

See [Secrets and fork PRs](../guides/secrets-and-fork-prs.md).

## Where Are Artifacts Written?

DiffPal writes artifacts under `.artifacts/diffpal/` by default. The canonical
findings bundle is `.artifacts/diffpal/findings.json`. Host commands can also
write `summary.md`, SARIF, Code Quality, and host-specific publishing payloads.

See [Artifacts](../reference/artifacts.md).

Next step: use [Getting Started](../getting-started/README.md) when you are
ready to configure the first review.
