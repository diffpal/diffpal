# GitLab CI

Merge a DiffPal job into `.gitlab-ci.yml` or the existing included CI structure. Preserve stages, includes, defaults, and unrelated jobs.

## Shared contract

- Run for merge-request pipelines and set `GIT_DEPTH: "0"`.
- Pass base from `CI_MERGE_REQUEST_DIFF_BASE_SHA`, head from `CI_COMMIT_SHA`, repository from `CI_PROJECT_PATH`, and a stable review ID using `CI_MERGE_REQUEST_IID`.
- Install the selected DiffPal CLI and configured provider with deliberate versions that follow repository pinning policy.
- Run `<diffpal-command> --profile ci doctor --mode local` with the same executable before review.
- Retain `.artifacts/diffpal/` with `artifacts: when: always` while preserving the review exit status. Register Code Quality or SARIF reports only when the job actually produces those files.
- Use a merge-request-scoped `resource_group` when concurrent jobs could duplicate native feedback.

## Trust and configuration

A same-project rule rejects forks but does not make the source checkout trusted. `.config/diffpal/config.yaml` in the merge request can change a provider `cmd`. A secret-bearing job therefore needs an organization-approved actor/branch plus a protected environment/manual maintainer decision, or trusted config loaded from the target branch or an external location.

A safe policy may combine `CI_PIPELINE_SOURCE == "merge_request_event"`, matching source and target project paths, a protected `DIFFPAL_TRUSTED_REVIEW == "true"` decision, and `when: manual`; the protected decision must not be settable by contributor code. For trusted external config, place `diffpal/config.yaml` or `config.yaml` under the selected root, pass `--config-dir <trusted-config-root>` to doctor and review, and verify the file exists so lookup cannot fall back to merge-request-controlled workspace config. Do not run contributor-controlled install scripts before trust is established.

## Artifact-only mode

- Invoke `review local`; do not pass `CI_JOB_TOKEN` or `GITLAB_TOKEN` for publication.
- The provider credential is still required behind the trust guard.

```bash
<diffpal-command> --profile ci review local \
  --base "$CI_MERGE_REQUEST_DIFF_BASE_SHA" \
  --head "$CI_COMMIT_SHA" \
  --repo "$CI_PROJECT_PATH" \
  --review-id "gitlab-mr-$CI_MERGE_REQUEST_IID" \
  --feedback <summary-or-review> \
  --out .artifacts/diffpal/findings.json
```

Append `--gate` when gating is enabled.

Store findings and captured Markdown stdout only as job artifacts unless a separate, explicitly approved publication step consumes them.

## Native publishing mode

- Invoke `review gitlab` with the same explicit identity values.
- Pass `CI_JOB_TOKEN` or a deliberately selected `GITLAB_TOKEN` separately from the provider credential. Store user-managed credentials as protected and masked variables.
- Explain whether summary/status, Code Quality, SARIF, or file-level discussions will be emitted for the selected feedback mode.
- Make `allow_failure` match the chosen gate policy.

## Merge and handoff

Validate stage ordering, includes, rules, artifact declarations, protected-variable scope, and YAML syntax. Preview the trust predicate, config origin, secret names, publication surface, gate, and fork behavior. Do not create variables, run a manual job, push, or publish during setup.
