# Migrate from 0.1.x to v1

DiffPal v1 removes the local state directory created by earlier `diffpal init`
commands. Host reconciliation now reads the current pull request or merge request
directly, so ephemeral CI jobs do not need a shared database.

## Required changes

1. Remove `--state` from any `diffpal init` invocation.
2. Delete `.config/diffpal/state/` if an earlier release created it.
3. Remove a `state/` entry from `.config/diffpal/.gitignore` when that file has
   no other project-specific entries.
4. Regenerate starter configuration with `diffpal init` if you want the v1
   template set.

Deleting the old directory does not remove review history. GitHub, GitLab, and
Azure DevOps comments and threads remain the source of published review state.

For credentialed CI, pin the CLI to an exact v1 release during migration and
upgrade deliberately after verifying one repeated review on a test pull request.
