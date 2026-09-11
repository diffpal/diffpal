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

For one shared provider, define one provider entry and point the root plus each requested profile to its ID. Do not add `local` or `ci` merely to make the example symmetric. Preserve unrelated `runtime`, `diffpal`, and `profiles` keys during a merge.

## Initialization choices

Supported wizard setup names are `codex-api-key`, `codex-subscription`, `copilot-github-token`, `opencode-acp`, and `generic-acp`. A representative command is:

```bash
<diffpal-command> init --wizard --setup <setup-name> --platform <platform> --profile <requested-profile>
```

The wizard creates one selected profile, not the full two-profile model. For a new repository, preview the wizard command and the post-init YAML merge together. For an existing config, normally skip `init` and patch the parsed YAML directly. Do not use `--force` by default.

When an approved `npx` invocation is selected, use that same command prefix for init and doctor. Do not fall back to a nonexistent global `diffpal` binary.

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
- provider IDs and types for each requested profile;
- any install or `npx` command;
- credential variable names only;
- the exact `doctor` commands, using the selected executable, that will validate the requested profiles.
