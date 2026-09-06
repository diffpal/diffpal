---
name: review
description: Run a safe DiffPal review of repository changes. Use when reviewing a branch or diff with DiffPal, producing a local findings artifact, optionally publishing review feedback to GitHub, GitLab, or Azure DevOps, or interpreting DiffPal review exit codes.
---

# Review with DiffPal

Default to a provider-backed local review using the `local` profile. Publishing is a separate action.

## Prepare the review

1. Read repository instructions and inspect the working tree, `.config/diffpal/config.yaml`, the selected provider ID, and relevant CI or host context. Do not expose credential values.
2. Unless the user specifies otherwise, select:
   - profile: `local`
   - mode: `review local`
   - head: `HEAD`
   - output: `.artifacts/diffpal/findings.json`
   - feedback: `review`
   - gate: disabled
3. Resolve a meaningful base revision from explicit user input or repository context. Do not guess across ambiguous branches or shallow history.
4. Run provider-free checks first:

   ```bash
   diffpal --profile local doctor --mode local
   diffpal --profile local debug prompt --base <base> --head <head> --format text
   ```

   Use the chosen profile and matching doctor mode when the user requested another environment or a host-publishing mode.

## Approve and execute

Before calling the provider, show:

- the exact command and review mode;
- profile and resolved provider ID;
- base, head, and repository identity;
- output path, feedback mode, gate state, and blocking threshold;
- that the selected provider may receive repository content and diff context.

Obtain explicit approval for that provider call. A request to inspect, prepare, or debug a review is not approval to call the provider.

For the default local review, run the approved equivalent of:

```bash
diffpal --profile local review local \
  --base <base> \
  --head <head> \
  --repo <repository-id> \
  --feedback review \
  --out .artifacts/diffpal/findings.json
```

Add `--gate` only when the user wants findings to affect the process result. Do not add host publishing merely because the user asked for a review.

When the requested mode is `github`, `gitlab`, or `ado`, explain its publication surfaces and obtain distinct explicit approval to publish. Approval for the provider call alone is insufficient. Never read or print the host token. GitHub `--dry-run` avoids publication; GitLab and Azure do not support it.

## Interpret the result

Report the artifact path, feedback/publishing outcome, finding summary, and status according to the process exit code:

| Code | Interpretation | Next action |
| --- | --- | --- |
| `0` | Review completed; findings may still exist when gate is off. | Summarize output and artifacts. |
| `2` | Config, profile, validation, flags, auth, or diff context failed. | Correct the identified setup problem before another run. |
| `3` | Transient provider failure. | Offer at most one retry and require approval before another provider call. |
| `4` | Publishing or conversion failed. | Preserve completed artifacts; do not rerun or republish automatically. |
| `5` | Internal tooling or local output failed. | Diagnose locally; do not assume the review completed. |
| `10` | Review completed with blocking findings while gate was enabled. | Report blockers and artifacts; do not call this a tooling crash. |
| `130` | Review was interrupted or cancelled. | Stop unless the user explicitly asks to resume. |

Never retry a provider call, publish again, change secrets, commit, or push without the authorization appropriate to that action.
