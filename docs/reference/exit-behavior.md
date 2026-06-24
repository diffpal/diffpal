# Exit Behavior

DiffPal uses process exit status to distinguish completed review, blocked
review, setup problems, provider problems, publishing problems, and internal
tooling failures.

| Code | Meaning |
| --- | --- |
| `0` | Command completed successfully. Review may still contain findings when gate is disabled. |
| `2` | Configuration, profile, provider validation, CLI flag, platform auth, or diff context/setup failure. |
| `3` | Transient provider failure such as provider timeout or retryable provider error. |
| `4` | Publishing or conversion failure, including platform API publishing and SARIF conversion. |
| `5` | Internal tooling or local file/output failure. |
| `10` | Blocking findings detected and `--gate` was enabled. |
| `130` | Interrupted or cancelled by the process environment. |

## Success

Without `--gate`, a review command returns `0` when review, artifact writing,
and any requested publishing complete. Findings can still be present and can
still be marked `blocking` in artifacts and summaries.

## Blocking Findings

With `--gate`, DiffPal counts findings whose severity meets
`diffpal.gate.block_on` or `--block-on`. If any are blocking, the command
returns exit code `10` after review output is written and publishing has been
attempted.

## Setup Failure

Exit code `2` covers invalid config, missing config, invalid feedback mode,
invalid review channel, invalid instructions file, missing platform token, and
missing host context such as base/head/repository metadata.

Review scope resolution failures are setup/context failures and also return `2`.

## Provider Failure

Provider validation failures return `2`. Transient provider runtime failures
return `3`. Other provider/runtime failures return `5`.

## Publishing Failure

Host publishing failures return `4`. This includes platform API failures and
publishing file conversion errors. The `diffpal sarif` command also returns `4`
for failed input reading, SARIF conversion, or SARIF output writing.

## Behavior With And Without Gating

| Gate enabled | Blocking findings exist | Exit code | Feedback/artifacts |
| --- | --- | --- | --- |
| no | no | `0` | written/published |
| no | yes | `0` | written/published, may report blocking status in output |
| yes | no | `0` | written/published |
| yes | yes | `10` | written/published before the gate error |

Tooling failures return their failure code regardless of gate setting because
the review result is incomplete.
