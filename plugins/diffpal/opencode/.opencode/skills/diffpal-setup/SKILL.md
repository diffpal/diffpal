---
name: diffpal-setup
description: Install, initialize, repair, or validate DiffPal configuration for requested local and/or CI profiles while preserving repository policy and credentials.
---

# Set up DiffPal

Configure only the environments the user requested. Do not turn local-only setup into CI work or vice versa.

## Inspect and select

1. Read repository instructions and inspect `.config/diffpal/config.yaml`, `.diffpalignore`, `.config/diffpal/templates/`, package manifests, CI files, and existing DiffPal version evidence. Detect provider CLIs by executable name only.
2. Resolve the requested targets: `local`, `ci`, or both. Resolve a code-host platform only for a target that needs host configuration. Preserve every unrequested profile.
3. Select one exact DiffPal command prefix and retain it through init and validation:
   - installed binary: `diffpal`;
   - approved one-shot: `npx -y @diffpal/diffpal@<version>`;
   - approved persistent install: `npm install --global @diffpal/diffpal@<version>`, then `diffpal`;
   - repository build only when the user explicitly requests source-tree dogfooding.
4. Read [profile guidance](references/profiles.md). When adding or changing a provider, also read [provider guidance](references/setup-providers.md). Do not invent provider commands, models, credentials, or versions that repository evidence does not support.

## Propose and approve

5. Build a structural merge in which every selected `diffpal.provider` resolves to a root `runtime.providers` entry. Keep shared policy at root and preserve unrelated runtime, platform, profile, template, and ignore content.
6. Before writing, show:
   - the exact install and init commands, with the chosen version and command prefix;
   - the YAML diff and every requested profile/provider change;
   - every file init may create: config, `.diffpalignore`, and each template under `.config/diffpal/templates/`;
   - whether each path will be created, preserved, or changed;
   - credential environment or secret names only.
7. Obtain separate approval for network/install actions and repository writes. Approval for config does not authorize installation or network access. Never use `--force` or replace existing files without approval for those exact paths.

## Apply and validate

8. Apply only the approved merge. For existing config, patch it directly. For new config, remember that the wizard creates one selected profile; add another only when it was requested and included in the preview.
9. Run `doctor` for each requested or changed profile using the same approved command prefix, for example:

```bash
<diffpal-command> --profile local doctor --mode local
<diffpal-command> --profile ci doctor --mode local
```

Do not run an unrequested profile. Use a host doctor mode only when its credential environment is intentionally available. `doctor --mode local` validates configuration and executables but does not prove provider authentication or connectivity.
10. Report all changed files, selected provider IDs and types, executable/version used, validation results, and remaining authentication or secret-store work.

## Boundaries

- Never request, read, print, copy, or embed credential values. Provider and code-host authentication remain separate.
- Inspection and doctor do not authorize a provider review, publishing, secret changes, workflow dispatch, commit, or push.
- Do not claim successful setup when the approved executable differs from the one validated.
