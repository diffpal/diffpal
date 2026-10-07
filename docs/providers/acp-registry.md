# ACP Registry Agents

## When To Use This Provider

Use an ACP Registry agent when the agent you want is published in the official
[ACP Registry](https://agentclientprotocol.com/registry) and you want Norma
Runtime to resolve its distribution instead of maintaining the ACP launch
command yourself.

DiffPal supports the `registry_acp` integration contract. Registry inclusion
does not mean that DiffPal has certified an individual agent, its model output,
license, authentication flow, platform support, or security.

## Prerequisites

- A DiffPal config committed at `.config/diffpal/config.yaml`.
- Node.js and `npx` available on the runner for the default launch path.
- A registry ID selected from the
  [official registry](https://agentclientprotocol.com/registry).
- The selected agent's own noninteractive authentication and platform
  requirements configured in trusted CI.

## How The Agent Starts

With `registry_id` and no `cmd`, Norma Runtime runs:

```text
npx -y @baldaworks/acprun@0.1.6 run <registry-id>
```

`bridge_version` selects the `@baldaworks/acprun` version and defaults to the
runtime-tested version. The example pins it explicitly. The launcher reads the
registry entry and starts its declared npm, Python, or binary distribution.

The ACP Registry and agent releases change independently from DiffPal. Review
the selected entry and its upstream project before allowing it to execute in a
credentialed job. Pinning `bridge_version` pins the launcher, not necessarily
the registry metadata or every artifact selected by the entry. For a more
operator-controlled launch path, set an explicit pinned `cmd`.

## Minimal Configuration

Start from
[`examples/configs/acp-registry/config.yaml`](../../examples/configs/acp-registry/config.yaml):

```yaml
runtime:
  providers:
    registry-agent:
      type: registry_acp
      registry_acp:
        registry_id: amp-acp
        bridge_version: 0.1.6

diffpal:
  provider: registry-agent
```

`amp-acp` demonstrates a valid registry ID and configuration shape; it is not a
DiffPal compatibility endorsement. Replace it with the ID you evaluated.

The `registry_acp` block also accepts the shared ACP session fields `model`,
`model_config_id`, `reasoning_effort`, `reasoning_effort_config_id`, `mode`, and
`extra_args`. Only set option IDs and values advertised by the selected agent.

## Explicit Command Override

Set `cmd` when you need a preinstalled or more tightly pinned launcher.
`registry_id` may be omitted when `cmd` is present:

```yaml
runtime:
  providers:
    pinned-agent:
      type: registry_acp
      registry_acp:
        cmd: ["/opt/acp-agents/my-agent", "--stdio"]

diffpal:
  provider: pinned-agent
```

With `cmd`, Norma Runtime uses that argv instead of constructing the default
`acprun` command. The executable must start an ACP stdio server.

## Authentication In CI

Follow the selected agent's authoritative authentication documentation. Keep
its credentials in protected CI secrets; DiffPal does not translate or manage
them. Do not expose agent credentials to untrusted fork jobs or run a registry
agent against fork-controlled code in a secret-bearing environment.

## Validate And Smoke Test

Check configuration and the local launcher prerequisite without starting or
downloading the agent:

```bash
diffpal doctor --profile ci --mode local
```

For the default path, doctor verifies that `npx` is available. For `cmd`, it
checks the first argv entry. It does not prove registry availability,
authentication, platform compatibility, or the ACP handshake.

Run a provider-backed smoke review only on a trusted branch:

```bash
diffpal --profile ci review local \
  --base origin/main \
  --head HEAD \
  --feedback summary \
  --out .artifacts/diffpal/findings.json
```

The smoke review should start the selected ACP agent, complete the structured
review, write the findings bundle, and print a Markdown summary.

## Common Failures

- `registry_id` is missing or does not match an official registry entry.
- Node.js or `npx` is unavailable for the default path.
- The registry, package registry, or binary download endpoint is unavailable.
- The registry entry has no distribution for the runner OS or architecture.
- The selected agent requires interactive login or missing credentials.
- The selected model, mode, or session option ID is not advertised by the agent.
- An explicit `cmd` is missing, not executable, or does not start ACP over stdio.
- `diffpal.provider` does not match the registry provider ID.

## Authoritative Sources

- [ACP Registry](https://agentclientprotocol.com/registry)
- [ACP Registry source and format](https://github.com/agentclientprotocol/registry)
- [Norma Runtime](https://github.com/normahq/runtime)

Next step: choose and evaluate a registry entry, then run the trusted smoke
review before enabling it in pull-request CI.
