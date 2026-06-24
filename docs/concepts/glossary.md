# Glossary

## PR/MR

Pull request or merge request. DiffPal uses the term for a proposed code change
that can be reviewed before merge.

## Provider

The configured runtime entry selected by `diffpal.provider` from
`runtime.providers`.

## ACP

Agent Client Protocol. DiffPal can run ACP-compatible CLIs as review agents.

## Finding

A structured review issue with severity, category, impact, message, and
location metadata.

## Publisher

The DiffPal component that turns validated review output into host-native
feedback such as comments, discussions, statuses, SARIF, or Code Quality
reports.

## Feedback

The visible review output users see in a host or local report, such as a
summary, inline comment, discussion, or thread.

## Gate

The optional CI behavior that returns a failing result when findings meet or
exceed the configured blocking threshold.

## Profile

A named config override selected for a run, commonly `ci`, that adjusts the
base repository config without changing the whole file.

## Artifact

A file written by DiffPal, usually under `.artifacts/diffpal/`, that records
the review result for CI uploads, auditing, or downstream processing.

## Base

The revision used as the comparison start. In PR/MR review, it is usually the
target branch revision or a merge base.

## Head

The revision being reviewed. In PR/MR review, it is usually the source branch
commit.

## Merge Base

The common ancestor used to compare a source branch against a target branch
when the host or CI system does not provide an exact base commit.
