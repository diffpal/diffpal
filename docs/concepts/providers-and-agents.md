# Providers And Agents

DiffPal separates provider selection from CI and code host setup.

## Provider Versus Agent

A provider is the configured runtime entry under `runtime.providers`. It tells
DiffPal which review runtime to use and which provider-specific settings belong
to that runtime.

An agent is the actual process that performs model-backed review work. For
ACP-based providers, the agent is a CLI process that starts an ACP stdio server.
For hosted provider types, the runtime talks to the hosted API described by the
provider config.

DiffPal sends the review task and validates the response. The provider or agent
owns model behavior, account access, provider tools, sandbox behavior, and
credentials.

## Built-In Provider Type Versus Generic ACP

Built-in provider types encode the known config shape for supported runtimes,
such as Codex ACP, Copilot ACP, and OpenCode ACP. They keep common setup
predictable and let generated configs use stable provider IDs.

`generic_acp` is the escape hatch for any CLI that can start an ACP stdio
server. DiffPal does not need provider-specific behavior for that path; it only
needs the command that starts the ACP-compatible agent.

Use [Providers](../providers/README.md) for setup pages and
[Configuration](../reference/configuration.md) for the exhaustive config
contract.

## Authentication Ownership

DiffPal does not create, license, or manage third-party provider accounts.
Provider authentication belongs to the provider and to your CI secret
management. Install and authenticate the selected provider before the DiffPal
review step, keep provider credentials out of untrusted fork jobs, and avoid
committing secrets to `.config/diffpal/config.yaml`.

Platform publishing credentials are separate from provider credentials. A CI
job may need both: one credential lets the provider perform review reasoning,
and another lets DiffPal publish feedback to GitHub, GitLab, or Azure DevOps.
