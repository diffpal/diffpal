# GitLab CI

Merge a diffpal-review job into .gitlab-ci.yml or the repository's existing included CI structure. Preserve current stages, includes, defaults, and unrelated jobs.

## Required shape

- Run for merge-request pipelines only after a trusted same-project and maintainer-controlled decision. The checked-in safe rule requires all of:
  - CI_PIPELINE_SOURCE equals merge_request_event
  - CI_MERGE_REQUEST_SOURCE_PROJECT_PATH equals CI_PROJECT_PATH
  - DIFFPAL_TRUSTED_REVIEW equals true
  - when: manual
- Set GIT_DEPTH: "0" so base and head are available.
- Install DiffPal and the configured provider with deliberately selected versions. For the checked-in Codex API-key recipe, current verified packages are @diffpal/diffpal@latest, @openai/codex@0.139.0, and @normahq/codex-acp-bridge@1.8.4; replace latest with a pinned DiffPal release when repository policy requires reproducibility.
- Invoke:

      diffpal --profile ci review gitlab

  Supply base from CI_MERGE_REQUEST_DIFF_BASE_SHA, head from CI_COMMIT_SHA, repo from CI_PROJECT_PATH, and a review ID using CI_MERGE_REQUEST_IID. Set feedback and gate explicitly.
- Keep the provider credential such as OPENAI_API_KEY separate from host publishing through CI_JOB_TOKEN or GITLAB_TOKEN. Store them as protected/masked CI variables and never embed values.
- Retain .artifacts/diffpal/ with artifacts: when: always; register codequality.json and diffpal.sarif reports when produced.
- Use a merge-request-scoped resource_group if concurrent review jobs could publish duplicate feedback.

## Merge and preview checks

Confirm that stage ordering and includes remain valid, the trust variable cannot be set by untrusted fork code, and allow_failure matches the chosen gate policy. Explain that feedback: review can create discussions/status output while summary is narrower.

Do not create protected variables, run the manual job, push, or publish. Provide those actions as separate maintainer steps after local YAML validation.
