#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"

make_fake_diffpal() {
  local bin_dir="$1"
  cat > "$bin_dir/diffpal" <<'SCRIPT'
#!/usr/bin/env bash
printf '%s\n' "$@" > "$DIFFPAL_ARGV_FILE"
SCRIPT
  chmod +x "$bin_dir/diffpal"
}

assert_contains() {
  local file="$1"
  local expected="$2"
  if ! grep -Fxq -- "$expected" "$file"; then
    echo "expected $file to contain: $expected" >&2
    echo "--- $file ---" >&2
    cat "$file" >&2
    exit 1
  fi
}

assert_not_contains() {
  local file="$1"
  local unexpected="$2"
  if grep -Fxq -- "$unexpected" "$file"; then
    echo "expected $file not to contain: $unexpected" >&2
    echo "--- $file ---" >&2
    cat "$file" >&2
    exit 1
  fi
}

test_installer_installs_requested_version() {
  local dir="$1"
  local fake_bin="$dir/fake-bin"
  local runner_temp="$dir/runner"
  local github_env="$dir/github-env"
  local npm_args="$dir/npm-args"
  mkdir -p "$fake_bin" "$runner_temp"

  cat > "$fake_bin/npm" <<'SCRIPT'
#!/usr/bin/env bash
printf '%s\n' "$@" > "$NPM_ARGV_FILE"
prefix=""
while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --prefix)
      prefix="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
mkdir -p "$prefix/bin"
cat > "$prefix/bin/diffpal" <<'BIN'
#!/usr/bin/env bash
exit 0
BIN
chmod +x "$prefix/bin/diffpal"
SCRIPT
  chmod +x "$fake_bin/npm"

  PATH="$fake_bin:$PATH" \
    RUNNER_TEMP="$runner_temp" \
    GITHUB_ENV="$github_env" \
    NPM_ARGV_FILE="$npm_args" \
    INPUT_INSTALL=true \
    INPUT_DIFFPAL_VERSION=0.1.2 \
    INPUT_DIFFPAL_PATH=diffpal \
    "$repo_root/tasks/github/install-diffpal.sh"

  assert_contains "$npm_args" "@diffpal/diffpal@0.1.2"
  assert_contains "$github_env" "DIFFPAL_BIN=$runner_temp/diffpal-action/bin/diffpal"
}

test_installer_selects_custom_path() {
  local dir="$1"
  local github_env="$dir/github-env-custom"
  mkdir -p "$dir"
  : > "$github_env"

  GITHUB_ENV="$github_env" \
    INPUT_INSTALL=true \
    INPUT_DIFFPAL_VERSION=latest \
    INPUT_DIFFPAL_PATH=/opt/diffpal/bin/diffpal \
    "$repo_root/tasks/github/install-diffpal.sh"

  assert_contains "$github_env" "DIFFPAL_BIN=/opt/diffpal/bin/diffpal"
}

test_installer_selects_path_when_install_disabled() {
  local dir="$1"
  local github_env="$dir/github-env-disabled"
  mkdir -p "$dir"
  : > "$github_env"

  GITHUB_ENV="$github_env" \
    INPUT_INSTALL=false \
    INPUT_DIFFPAL_VERSION=latest \
    INPUT_DIFFPAL_PATH=diffpal \
    "$repo_root/tasks/github/install-diffpal.sh"

  assert_contains "$github_env" "DIFFPAL_BIN=diffpal"
}

test_wrapper_uses_installed_binary_and_feedback() {
  local dir="$1"
  local bin_dir="$dir/installed/bin"
  local argv="$dir/argv-installed"
  mkdir -p "$bin_dir"
  make_fake_diffpal "$bin_dir"

  DIFFPAL_ARGV_FILE="$argv" \
    DIFFPAL_BIN="$bin_dir/diffpal" \
    INPUT_BASE=base-sha \
    INPUT_HEAD=head-sha \
    INPUT_FEEDBACK=summary \
    INPUT_REVIEW_CHANNEL=diffpal-dev \
    "$repo_root/tasks/github/run-diffpal-review.sh"

  assert_contains "$argv" "review"
  assert_contains "$argv" "github"
  assert_contains "$argv" "--feedback"
  assert_contains "$argv" "summary"
  assert_contains "$argv" "--review-channel"
  assert_contains "$argv" "diffpal-dev"
}

test_wrapper_mode_overrides_feedback() {
  local dir="$1"
  local bin_dir="$dir/path-bin"
  local argv="$dir/argv-mode"
  mkdir -p "$bin_dir"
  make_fake_diffpal "$bin_dir"

  PATH="$bin_dir:$PATH" \
    DIFFPAL_ARGV_FILE="$argv" \
    DIFFPAL_BIN=diffpal \
    INPUT_BASE=base-sha \
    INPUT_HEAD=head-sha \
    INPUT_MODE=check-run \
    INPUT_FEEDBACK=summary \
    "$repo_root/tasks/github/run-diffpal-review.sh"

  assert_contains "$argv" "--mode"
  assert_contains "$argv" "check-run"
  assert_not_contains "$argv" "--feedback"
  assert_not_contains "$argv" "summary"
}

test_wrapper_can_disable_summary_overview() {
  local dir="$1"
  local bin_dir="$dir/overview-bin"
  local argv="$dir/argv-overview"
  mkdir -p "$bin_dir"
  make_fake_diffpal "$bin_dir"

  PATH="$bin_dir:$PATH" \
    DIFFPAL_ARGV_FILE="$argv" \
    DIFFPAL_BIN=diffpal \
    INPUT_BASE=base-sha \
    INPUT_HEAD=head-sha \
    INPUT_SUMMARY_OVERVIEW=false \
    "$repo_root/tasks/github/run-diffpal-review.sh"

  assert_contains "$argv" "--summary-overview=false"
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

test_installer_installs_requested_version "$tmpdir/install"
test_installer_selects_custom_path "$tmpdir/custom"
test_installer_selects_path_when_install_disabled "$tmpdir/disabled"
test_wrapper_uses_installed_binary_and_feedback "$tmpdir/wrapper-installed"
test_wrapper_mode_overrides_feedback "$tmpdir/wrapper-mode"
test_wrapper_can_disable_summary_overview "$tmpdir/wrapper-overview"

echo "github action smoke tests passed"
