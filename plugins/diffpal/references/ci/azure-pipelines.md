# Azure Pipelines

Merge DiffPal into the selected Azure pipeline without replacing existing triggers, pools, stages, jobs, or steps.

## Shared contract

- Use `checkout: self` with `fetchDepth: 0`.
- Map the pull request's target/base commit, source/head commit, repository ID, and pull-request ID explicitly when using the CLI. The native task may derive these from Azure variables.
- Install the selected DiffPal CLI and configured provider with deliberate versions that follow repository pinning policy.
- Run `<diffpal-command> --profile ci doctor --mode local` with the same executable before review.
- Publish `.artifacts/diffpal/` with `PublishPipelineArtifact@1` under `condition: always()`, while preserving the DiffPal exit status.

## Trust and configuration

The common condition below filters forks, but does not establish trust in the checked-out source:

```yaml
condition: and(succeeded(), ne(variables['System.PullRequest.IsFork'], 'True'))
```

A pull request can alter `.config/diffpal/config.yaml`, including a provider `cmd`. Before exposing provider credentials, require an organization-approved author/branch plus an Environment approval or other maintainer-controlled gate, or load config from the trusted target branch or an external secured location. Put `diffpal/config.yaml` or `config.yaml` under the chosen trusted root, pass `--config-dir <trusted-config-root>` to doctor and review, and verify the file exists so lookup cannot fall back to source-controlled config. Do not run source-controlled scripts before the trust gate.

## Artifact-only mode

- Invoke `review local` with explicit base, head, repo, review ID, feedback, output, and optional gate.
- Do not enable OAuth token access or pass `$(System.AccessToken)` to DiffPal.
- Expose only the provider credential after the trust guard; retain findings and captured Markdown stdout as pipeline artifacts.

## Native publishing mode

- Use `DiffPalReview@1` with deliberate `diffpalVersion`, `profile: ci`, explicit feedback, and explicit gate, or invoke `review ado` with explicit metadata.
- Enable and pass `$(System.AccessToken)` only to the publishing step, separately from the provider credential.
- If the task cannot receive the required trusted `--config-dir`, use the CLI path instead.
- Explain the selected summary/comment/status surfaces and whether the gate can fail a branch-policy check.

## Merge and handoff

Verify PR branch filters, variable-group authorization, Environment checks, conditions on every credentialed step, artifact publication, and YAML syntax. Preview the trust predicate, config origin, versions, secret names, OAuth permission, feedback, gate, and fork behavior. Do not create variable values, enable OAuth access, approve an Environment, push, queue, or publish during setup.
