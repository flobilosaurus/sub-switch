#!/bin/sh
set -eu

repo="${SUB_SWITCH_REPO:-flobilosaurus/sub-switch}"
version="${SUB_SWITCH_VERSION:-latest}"
install_dir="${INSTALL_DIR:-$HOME/.local/bin}"
binary_name="sub-switch"
base_url="${SUB_SWITCH_BASE_URL:-https://github.com/${repo}/releases}"

need_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "error: required command not found: $1" >&2
    exit 1
  fi
}

need_cmd curl
need_cmd uname
need_cmd mktemp

os=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$os" in
  linux|darwin) ;;
  *)
    echo "error: unsupported OS: $os" >&2
    exit 1
    ;;
esac

arch=$(uname -m)
case "$arch" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *)
    echo "error: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset="sub-switch-${os}-${arch}"
base_url=${base_url%/}
if [ "$version" = "latest" ]; then
  url="${base_url}/latest/download/${asset}"
else
  url="${base_url}/download/${version}/${asset}"
fi

tmp=$(mktemp "${TMPDIR:-/tmp}/sub-switch.XXXXXX")
cleanup() {
  rm -f "$tmp"
}
trap cleanup EXIT INT HUP TERM

echo "Downloading ${url}"
curl -L --silent --show-error --fail -o "$tmp" "$url"
chmod 755 "$tmp"

mkdir -p "$install_dir"
if ! mv "$tmp" "${install_dir}/${binary_name}"; then
  echo "error: failed to install to ${install_dir}/${binary_name}" >&2
  echo "hint: set INSTALL_DIR to a writable directory, for example:" >&2
  echo "      INSTALL_DIR=\$HOME/.local/bin sh -c '\$(curl -fsSL https://raw.githubusercontent.com/${repo}/main/install.sh)'" >&2
  exit 1
fi
trap - EXIT INT HUP TERM

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    echo "warning: ${install_dir} is not on PATH" >&2
    ;;
esac

"${install_dir}/${binary_name}" --version >/dev/null 2>&1 || true

echo "Installed ${binary_name} to ${install_dir}/${binary_name}"
