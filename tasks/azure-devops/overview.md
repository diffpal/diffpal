# DiffPal Review

DiffPal Review adds a pipeline task that runs DiffPal pull request review in
Azure DevOps.

DiffPal is a diff-first AI review tool. It collects the pull request diff,
delegates review to the configured provider, writes structured findings, and can
publish Azure DevOps pull request feedback.

## Usage

```yaml
steps:
  - task: DiffPalReview@1
    inputs:
      profile: ci
      feedback: balanced
      gate: true
    env:
      SYSTEM_ACCESSTOKEN: $(System.AccessToken)
      OPENAI_API_KEY: $(OPENAI_API_KEY)
```

The task installs the configured DiffPal CLI package, resolves pull request
base/head revisions from Azure Pipelines variables when inputs are omitted, and
passes provider credentials through environment variables.

## Requirements

- Enable scripts to access `System.AccessToken` for pull request publishing.
- Add the provider credential required by your DiffPal profile, such as
  `OPENAI_API_KEY`.
- Keep DiffPal configuration in `.config/diffpal/config.yaml` or pass a custom
  `configDir`.

## Documentation

- Repository: <https://github.com/diffpal/diffpal>
- Azure Pipelines setup guide: <https://github.com/diffpal/diffpal/blob/main/docs/ci-examples.md#azure-pipelines>
- Configuration reference: <https://github.com/diffpal/diffpal/blob/main/docs/config-reference.md>
- Provider and CI examples: <https://github.com/diffpal/diffpal/tree/main/examples>
