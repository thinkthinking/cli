#!/usr/bin/env bash
# thinkthinking 一键安装脚本（从 GitHub Releases 下载二进制）。
#
# 用法：
#   curl -fsSL https://raw.githubusercontent.com/thinkthinking/cli/master/scripts/install.sh | bash
#
# 环境变量：
#   THINKTHINKING_VERSION   指定版本（默认 latest）
#   THINKTHINKING_INSTALL_DIR  安装目录（默认 /usr/local/bin）

set -euo pipefail

REPO="thinkthinking/cli"
BINARY="thinkthinking"
INSTALL_DIR="${THINKTHINKING_INSTALL_DIR:-/usr/local/bin}"

# 探测 os / arch，对齐 GoReleaser 命名。
detect_platform() {
  local os arch
  case "$(uname -s)" in
    Darwin) os="darwin" ;;
    Linux)  os="linux" ;;
    *) echo "unsupported OS: $(uname -s)" >&2; exit 1 ;;
  esac
  case "$(uname -m)" in
    x86_64|amd64) arch="amd64" ;;
    arm64|aarch64) arch="arm64" ;;
    *) echo "unsupported arch: $(uname -m)" >&2; exit 1 ;;
  esac
  echo "${os}_${arch}"
}

# 解析版本（默认取 latest release tag）。
resolve_version() {
  if [ -n "${THINKTHINKING_VERSION:-}" ]; then
    echo "${THINKTHINKING_VERSION#v}"
    return
  fi
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed -E 's/.*"v?([^"]+)".*/\1/'
}

main() {
  local platform version archive url tmp
  platform="$(detect_platform)"
  version="$(resolve_version)"
  if [ -z "$version" ]; then
    echo "failed to resolve version (no releases yet?)" >&2
    exit 1
  fi

  archive="${BINARY}_${version}_${platform}.tar.gz"
  url="https://github.com/${REPO}/releases/download/v${version}/${archive}"
  tmp="$(mktemp -d)"

  echo "downloading ${url}"
  curl -fsSL "$url" -o "${tmp}/${archive}"
  tar -xzf "${tmp}/${archive}" -C "$tmp"

  echo "installing to ${INSTALL_DIR}/${BINARY}"
  if [ -w "$INSTALL_DIR" ]; then
    mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  else
    sudo mv "${tmp}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  fi
  chmod +x "${INSTALL_DIR}/${BINARY}"
  rm -rf "$tmp"

  echo "installed: $(${INSTALL_DIR}/${BINARY} version)"
}

main "$@"
