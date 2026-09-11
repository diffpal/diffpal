# Native uncommitted review

Use the dedicated command from the repository whose current changes should be
reviewed:

```bash
npx -y @diffpal/diffpal@latest --profile local review uncommitted \
  --feedback review --out .artifacts/diffpal/findings.json
```

The CLI selects an uncommitted-review task. The backend provider obtains and
inspects its workspace snapshot with its own tools. Do not enumerate changed
paths, pass `--path` or `--untracked`, create a temporary repository, stage,
commit, or stash files as preparation. The command does not accept `--base` or
`--head` because this mode is not a committed revision-range review.

Use the configured `local` profile unless the user selected another profile.
Add `--gate` only when requested. Existing authorization to run the review is
sufficient; do not ask again unless the scope, provider, or external effects
change. Local uncommitted review writes the normal findings bundle and prints
Markdown; it does not publish to a code host.

If the backend cannot inspect its workspace, report the provider error. Do not
fall back to a CLI-created snapshot or a file-by-file prompt. If the latest
published CLI does not expose `review uncommitted`, report the release gap
instead of building repository source implicitly.
