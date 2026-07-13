#!/usr/bin/env bash
# Installs the msdl CLI. Usage: curl -fsSL https://api.msdl.tech-latest.com/install.sh | bash
set -euo pipefail

REPO="starkSV/windows-iso-downloader"

os="$(uname -s)"
arch="$(uname -m)"

case "$os" in
  Linux) platform="linux" ;;
  Darwin) platform="darwin" ;;
  *)
    echo "msdl: unsupported OS: $os (Windows users: winget install starkSV.msdl)" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64|amd64) goarch="amd64" ;;
  aarch64|arm64) goarch="arm64" ;;
  *)
    echo "msdl: unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

asset="msdl-${platform}-${goarch}"
url="https://github.com/${REPO}/releases/latest/download/${asset}"

tmp="$(mktemp)"
echo "Downloading ${asset}..."
curl -fsSL "$url" -o "$tmp"
chmod +x "$tmp"

if [ -n "${PREFIX:-}" ] && [ -d "${PREFIX}/bin" ]; then
  # Termux (or similar $PREFIX-based environment) -- no sudo available or needed
  mv "$tmp" "${PREFIX}/bin/msdl"
  echo "Installed to ${PREFIX}/bin/msdl"
elif [ -w /usr/local/bin ]; then
  mv "$tmp" /usr/local/bin/msdl
  echo "Installed to /usr/local/bin/msdl"
else
  sudo mv "$tmp" /usr/local/bin/msdl
  echo "Installed to /usr/local/bin/msdl"
fi

echo "Run 'msdl --help' to get started."
