# Committed review scopes

Use `review local` with explicit stable revisions for a branch, PR, range or commit. Uncommitted changes use the separate native mode linked from the skill.

| Intent | Base | Head |
| --- | --- | --- |
| PR/branch | Explicit target or verified remote-tracking target | HEAD or explicit source SHA |
| Explicit range | User-provided base | User-provided head |
| One non-merge commit | Commit parent | Commit |

Resolve the target from user input, PR metadata or canonical remote information; do not infer it from branch naming alone. Preserve range order. For merge commits, resolve the intended parent. A root commit requires an explicit supported base; do not silently select another commit.

```bash
git rev-parse --verify '<base>^{commit}'
git rev-parse --verify '<head>^{commit}'
git diff --name-status <base>..<head>
<diffpal> --profile local debug prompt \
  --base <base> --head <head> --out .artifacts/diffpal/preview.json
<diffpal> --profile local review local \
  --base <base> --head <head> --repo <repository-id> \
  --feedback review --out .artifacts/diffpal/findings.json
```

A branch range excludes dirty changes. If the source checkout contains excluded changes or is not at the chosen head, use an isolated detached worktree at the committed head for provider inspection. Include creation and cleanup in the authorized local preparation. Keep artifact output absolute when retaining it in the original repository.

Load config from its intended source explicitly when needed: for the standard source layout, `--config-dir <source-root>/.config` searches `diffpal/config.yaml` there. Verify the config path rather than relying on fallback. Never remove the review workspace while its provider is running.
