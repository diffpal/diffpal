# Azure Pipelines

Merge DiffPal into the selected Azure pipeline without replacing existing triggers, pools, stages, jobs, or steps.

## Required shape

- Checkout self with fetchDepth: 0.
- Keep provider authentication and review steps behind:

      and(succeeded(), ne(variables['System.PullRequest.IsFork'], 'True'))

  Use a stricter organization-specific trusted-source condition when required.
- Install and authenticate the configured provider with deliberately selected versions. The checked-in Codex API-key recipe uses Node 22, @openai/codex@0.139.0, and @normahq/codex-acp-bridge@1.8.4.
- Use DiffPalReview@1 with profile: ci, explicit feedback, explicit gate, and a deliberate diffpalVersion. Pin the DiffPal version when repository policy requires reproducibility.
- The native task derives pull-request metadata from Azure variables. If using the CLI instead, pass base, head, repository ID, and review ID explicitly.
- Pass a provider secret such as OPENAI_API_KEY separately from the host publishing token $(System.AccessToken). Never place either value in YAML.
- Add PublishPipelineArtifact@1 for .artifacts/diffpal/ under an always() condition unless the current pipeline already retains these outputs.

## Merge and preview checks

Verify that PR triggers target the intended branches, scripts receive secrets through env, OAuth access is enabled only when native publishing is selected, and the fork condition covers every credentialed provider and review step. Explain the selected feedback surfaces and whether the gate can fail a branch-policy check.

Do not create variable-group values, enable OAuth access, push, queue the pipeline, or publish. Report those as explicit Azure administrator or maintainer steps.
