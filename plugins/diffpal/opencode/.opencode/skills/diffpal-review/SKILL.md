---
name: diffpal-review
description: Run and verify DiffPal reviews for branches, revision ranges, commits, and uncommitted work; optionally publish committed reviews to a code host.
---

# Review with DiffPal

Run the review mode the user requested. Local uncommitted review and committed
revision review are separate modes; never combine them implicitly.

## Select the executable

Honor an executable or version explicitly selected by the user. Otherwise use
the latest published CLI:

```bash
npx -y @diffpal/diffpal@latest
```

Reuse that exact prefix throughout the run. Do not build repository source or
prefer a stale global binary unless the user explicitly requests development
testing or that binary.

## Select the review mode

- **Uncommitted workspace:** use `review uncommitted` and read
  [uncommitted review](references/uncommitted-changes.md). The backend
  provider owns workspace snapshot acquisition and inspection; the CLI mode
  changes the task prompt. Do not enumerate or pass changed files one by one.
- **Committed branch, range, or commit:** use `review local --base … --head …`
  and read [committed scopes](references/local-review-scopes.md).
- **Host publication:** use the matching committed `review github`,
  `review gitlab`, or `review ado` mode only after distinct authorization to
  publish.

When the user says to review current or uncommitted work, choose the
uncommitted command directly. Ask a scope question only when committed history
and current workspace changes are both plausible and the user did not choose.

## Run

Read repository instructions and the selected profile from
`.config/diffpal/config.yaml` without exposing credentials. Use `local` for a
developer review unless the user selected another profile. A prior explicit
request to run the review authorizes the provider call; ask again only when the
provider, scope, publishing behavior, or other external effect changes.

For uncommitted work, run the single command documented in the reference. Do
not require a provider-free file preview: the backend obtains its snapshot and
uses its own repository tools. For committed work, resolve and verify explicit
base/head revisions before running the reference command.

Default feedback to `review`, output to
`.artifacts/diffpal/findings.json`, and gate off. Add `--gate` only when the
user wants findings to control the exit status. Local review never publishes.

## Verify and report

Confirm the findings artifact exists and its base/head, reviewed files, change
summary, and findings are consistent with the selected mode. Do not treat exit
0 alone as proof that the requested scope was reviewed. Report the artifact
path, concise finding summary, and any provider or validation failure.

| Exit | Meaning |
| --- | --- |
| 0 | Review completed; findings may exist when gate is off. |
| 2 | Configuration, flags, authentication, or diff context failed. |
| 3 | Transient provider failure; offer at most one authorized retry. |
| 4 | Publishing or conversion failed; preserve completed artifacts. |
| 5 | Internal tooling or local output failed. |
| 10 | Review completed with gated blocking findings. |

Never retry a provider call indefinitely, publish, change secrets, stage,
commit, stash, push, or construct a replacement snapshot without the authority
appropriate to that action.
