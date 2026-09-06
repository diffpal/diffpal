---
name: diffpal-setup
description: Configure DiffPal in a repository with safe local and CI profiles. Use when installing or initializing DiffPal, changing providers, adding a local or ci profile, repairing .config/diffpal/config.yaml, or validating DiffPal setup.
---

# Set up DiffPal

Configure DiffPal without discarding repository-owned policy or credentials.

## Workflow

1. Read repository instructions and inspect, when present:
   - `.config/diffpal/config.yaml`, `.diffpalignore`, and `.config/diffpal/templates/`
   - package manifests and existing DiffPal installation/version evidence
   - available provider CLIs by executable name only
2. Determine the requested install path, code-host platform, and provider for each of `local` and `ci`. They may share one provider ID or use different IDs.
3. Read [profile guidance](../../references/profiles.md) before proposing configuration.
4. Validate the proposal structurally:
   - every selected `diffpal.provider` names an entry in `runtime.providers` after its profile is applied;
   - shared review policy remains at root unless an environment needs an override;
   - existing providers, profiles, platform settings, templates, and ignore rules are preserved.
5. Show the exact install/init commands and YAML diff before changing anything. State separately which action would:
   - install software or run network-backed `npx -y`;
   - write repository configuration.
6. Obtain explicit approval for each applicable boundary. Do not treat approval to edit config as approval to install software or access the network.
7. Apply only the approved merge. Never use `diffpal init --force` or overwrite an existing file unless the user explicitly approves that exact overwrite.
8. Validate both profiles without making a provider review call:

   ```bash
   diffpal --profile local doctor --mode local
   diffpal --profile ci doctor --mode local
   ```

   Use the CI host mode as an additional check only when its credential environment is intentionally available.
9. Report changed files, selected provider IDs, validation results, and manual authentication or secret-store work still required.

## Safety boundaries

- Never request, read, print, copy, or embed credential values. Refer only to documented environment-variable or secret names.
- Provider authentication belongs to the provider; code-host authentication is separate.
- Inspection and `doctor` do not authorize a provider review call, publishing, secret changes, workflow dispatch, commit, or push.
- If DiffPal is unavailable, present supported choices such as a global install or one-shot `npx`; wait for explicit approval before executing either installation or network-backed resolution.
- `diffpal init --wizard` generates one selected profile. Do not claim it creates both `local` and `ci`; merge the second profile into the preview.
