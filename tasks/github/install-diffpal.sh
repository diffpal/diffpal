#!/usr/bin/env bash
set -euo pipefail

truthy() {
  case "${1,,}" in
    1 | true | yes | y | on)
      return 0
      ;;
    *)
      return 1
      ;;
  esac
}

install_requested="${INPUT_INSTALL:-true}"
diffpal_path="${INPUT_DIFFPAL_PATH:-diffpal}"
version="${INPUT_DIFFPAL_VERSION:-latest}"

if [[ "$diffpal_path" != "diffpal" ]]; then
  echo "Using custom diffpal-path: $diffpal_path"
  echo "DIFFPAL_BIN=$diffpal_path" >> "$GITHUB_ENV"
  exit 0
fi

if ! truthy "$install_requested"; then
  echo "DiffPal installation disabled; using diffpal from PATH"
  echo "DIFFPAL_BIN=diffpal" >> "$GITHUB_ENV"
  exit 0
fi

if ! command -v npm >/dev/null 2>&1; then
  echo "npm is required to install @diffpal/diffpal but was not found on PATH" >&2
  exit 127
fi

install_root="${RUNNER_TEMP:-/tmp}/diffpal-action"
mkdir -p "$install_root"

npm install --global --prefix "$install_root" "@diffpal/diffpal@$version" --omit=dev --no-audit --no-fund

diffpal_bin="$install_root/bin/diffpal"
if [[ ! -x "$diffpal_bin" ]]; then
  echo "installed diffpal binary was not found: $diffpal_bin" >&2
  exit 127
fi

echo "DIFFPAL_BIN=$diffpal_bin" >> "$GITHUB_ENV"
echo "Installed DiffPal: $diffpal_bin"
