# DiffPal profile guidance

Use this reference while constructing or merging `.config/diffpal/config.yaml`.

## Merge model

DiffPal loads root configuration, applies the selected profile from `profiles`, then applies environment overrides. Keep every provider definition under root `runtime.providers`; profile-level `diffpal.provider` selects one of those provider IDs.

Keep policy shared at root when local and CI should behave alike. Put only intentional differences beneath `profiles.local.diffpal` and `profiles.ci.diffpal`. Both profiles may select the same provider.

```yaml
version: v1

runtime:
  providers:
    local-agent:
      type: codex_acp
      codex_acp:
        reasoning_effort: low
    ci-agent:
      type: codex_acp
      codex_acp:
        reasoning_effort: low

diffpal:
  provider: local-agent
  gate:
    block_on: high
  review:
    language: en
    instructions: |
      Prefer actionable findings supported directly by the diff.

profiles:
  local:
    diffpal:
      provider: local-agent
  ci:
    diffpal:
      provider: ci-agent
```

For one shared provider, define one provider entry and point the root, `local`, and `ci` selections to its ID. Preserve unrelated `runtime`, `diffpal`, and `profiles` keys during a merge.

## Initialization choices

Supported wizard setup names are `codex-api-key`, `codex-subscription`, `copilot-github-token`, `opencode-acp`, and `generic-acp`. A representative command is:

```bash
diffpal init --wizard --setup <setup-name> --platform <platform> --profile ci
```

The wizard creates one selected profile, not the full two-profile model. For a new repository, preview the wizard command and the post-init YAML merge together. For an existing config, normally skip `init` and patch the parsed YAML directly. Do not use `--force` by default.

When only `npx` is available, the equivalent command begins with `npx -y @diffpal/diffpal@latest`; this can resolve and execute code from the network and needs explicit approval.

## Provider and credential rules

- Use only documented provider types: `generic_acp`, `gemini_acp`, `codex_acp`, `opencode_acp`, `copilot_acp`, `claude_code_acp`, `openai`, `aistudio`, or `pool`.
- The effective `diffpal.provider` must match an effective `runtime.providers` key.
- Prefer environment variables or CI secret references over credential values in YAML.
- Never reveal or transfer credentials between the provider and the code host.
- Use `--profile local` for developer runs and `--profile ci` for automation. `DIFFPAL_PROFILE` is acceptable when a CI runner sets it deliberately.
- Keep `--feedback` and `--gate` explicit at review time; a profile should not obscure whether feedback will publish or whether findings will fail a job.

## Preview checklist

Before writing, show:

- the target config path and whether it already exists;
- the preserved and changed YAML keys;
- local and CI provider IDs and types;
- any install or `npx` command;
- credential variable names only;
- the exact `doctor` commands that will validate the result.
