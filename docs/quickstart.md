# DiffPal Quickstart

## Install

### Local

```bash
npm install @diffpal/diffpal@latest
npx diffpal version
npx diffpal doctor
```

Go source installs remain available for development:

```bash
go install github.com/diffpal/diffpal/cmd/diffpal@latest
diffpal version
diffpal doctor
```

### GitHub Action

```yaml
- uses: actions/setup-node@v4
  with:
    node-version: 20
- run: npm install @diffpal/diffpal@latest
- uses: diffpal/action@v1
  with:
    diffpal-path: ./node_modules/.bin/diffpal
    base: ${{ github.event.pull_request.base.sha }}
    head: ${{ github.event.pull_request.head.sha }}
    gate: true
```

## First run

1. Initialize configuration files in a repository:

```bash
diffpal init
```

2. Review a local diff:

```bash
diffpal review local \
  --base origin/main \
  --head HEAD \
  --context-lines 20 \
  --max-files 200 \
  --out .artifacts/diffpal/findings.json
```

3. Check local runtime and configuration:

```bash
diffpal doctor
```

4. Produce machine output:

```bash
diffpal sarif --input .artifacts/diffpal/findings.json --out .artifacts/diffpal/diffpal.sarif
```

5. Run a host-scoped CI review:

```bash
diffpal review github \
  --base "${BASE_SHA}" \
  --head "${HEAD_SHA}" \
  --block-on high \
  --gate
```

Host review modes require platform auth through config values or standard CI
environment variables such as `GITHUB_TOKEN`. Envsubst placeholders such as
`token: "${GITHUB_TOKEN}"` remain supported, but missing referenced variables
fail config load before the command can run.

Equivalent host modes exist for GitLab and Azure DevOps:

```bash
diffpal review gitlab --base "${BASE_SHA}" --head "${HEAD_SHA}" --gate
diffpal review ado --base "${BASE_SHA}" --head "${HEAD_SHA}" --gate
```

## Common flags

`review local` accepts:

- `--base`: base commit SHA or ref
- `--head`: head commit SHA or ref
- `--max-files`: hard diff file limit
- `--context-lines`: changed-line neighborhood context
- `--block-on`: mark findings at this severity and above as blocking
- `--gate`: exit 1 with `review blocked` when blocking findings are present

`review github|gitlab|ado` additionally accept:

- `--mode`: override the selected host's default publish surfaces

All commands emit structured finding bundles to:

`.artifacts/diffpal/findings.json`

Starter config is written to:

`.config/diffpal/config.yaml`
