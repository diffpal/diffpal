# DiffPal coding-agent plugin

This instruction-only plugin helps coding agents set up DiffPal, run reviews, and add safe CI automation. Version 0.1.0 is published by Alexey Samoylov under the MIT License.

## Skills

| Purpose | Codex | Claude Code | Grok, Copilot, Cursor, OpenCode |
| --- | --- | --- | --- |
| Configure DiffPal and local/CI profiles | $diffpal:setup | /diffpal:setup | diffpal-setup |
| Run a local or explicitly published review | $diffpal:review | /diffpal:review | diffpal-review |
| Configure GitHub, GitLab, Azure, or custom CI | $diffpal:ci | /diffpal:ci | diffpal-ci |

Agent Plugins 1.0.0 clients discover setup, review, and ci from the package's skills directory. The portable standard does not define a marketplace, installation command, or invocation namespace; follow the client's own UI and invocation rules.

## Install or distribute

The commands below modify the selected host's plugin configuration and may clone this repository. Review the current host prompt before accepting.

### Codex

    codex plugin marketplace add diffpal/diffpal
    codex plugin add diffpal@diffpal

Invoke $diffpal:setup, $diffpal:review, or $diffpal:ci.

### Claude Code

    claude plugin marketplace add diffpal/diffpal
    claude plugin install diffpal@diffpal

Invoke /diffpal:setup, /diffpal:review, or /diffpal:ci.

### Grok Build

Register the marketplace for discovery, or install the package subdirectory directly:

    grok plugin marketplace add diffpal/diffpal
    grok plugin install diffpal/diffpal#plugins/diffpal

Use the flat skill names diffpal-setup, diffpal-review, and diffpal-ci.

### GitHub Copilot CLI

    copilot plugin marketplace add diffpal/diffpal
    copilot plugin install diffpal@diffpal

Use the flat skill names diffpal-setup, diffpal-review, and diffpal-ci.

### Cursor

Register the Git repository, then select DiffPal in Cursor's marketplace UI:

    agent plugin marketplace add https://github.com/diffpal/diffpal

Use the flat skill names diffpal-setup, diffpal-review, and diffpal-ci.

### OpenCode

OpenCode distribution is a manual, project-local copy, not an npm plugin install. From a clone of this repository, copy the prepared native skill directories into the target project:

    mkdir -p /path/to/project/.opencode/skills
    cp -R plugins/diffpal/opencode/.opencode/skills/diffpal-* /path/to/project/.opencode/skills/

Open the target project with OpenCode and request diffpal-setup, diffpal-review, or diffpal-ci by skill name.

### Agent Plugins 1.0.0

Point a compatible client at plugins/diffpal according to that client's package-loading instructions. The root plugin.json is the portable manifest. Do not infer an installation or invocation command from the Agent Plugins specification.

## Profiles

DiffPal repository configuration remains at .config/diffpal/config.yaml. Shared providers belong under runtime.providers and shared review policy under diffpal. The local and ci profiles select the provider appropriate to each environment:

- developer runs use --profile local;
- automated runs use --profile ci;
- both profiles may select the same provider or different provider IDs.

The setup skill previews a merge and preserves unrelated configuration. DiffPal's wizard initializes one selected profile, so the skill adds the second profile explicitly rather than claiming the wizard creates both.

## Effects and security

- setup can propose installation and repository-config edits, but requires separate approval for network/install actions and writes. It never uses --force implicitly.
- review performs provider-free preflight checks, then requires approval before repository content is sent to a provider. Host publishing needs distinct approval.
- ci edits workflow files only after preview. It separates provider credentials from code-host credentials and requires trusted-source or fork protection.
- No skill reads, prints, embeds, or sets credential values. Installing tools, changing secrets, pushing, dispatching CI, and publishing are outside ordinary plugin use unless separately authorized.

## Requirements and validation

Actual review execution requires DiffPal, a configured provider, a resolvable Git base/head, and provider authentication. Native publishing additionally requires the matching code-host context and token.

The package is statically validated for Codex, Claude Code, Grok Build, Agent Skills frontmatter, Agent Plugins 1.0.0, paths, and metadata. Copilot CLI has no dedicated validate command; Cursor requires a load session for runtime confirmation; OpenCode project-local discovery requires opening a target project. Static validation does not claim that any host installed, loaded, or invoked the plugin.

This package includes skills only: no hooks, commands, agents, MCP servers, LSP servers, apps, or executable plugin code.
