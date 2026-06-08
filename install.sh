#!/usr/bin/env sh
set -eu

REPO="${ACME233_REPO:-neko233-com/acme233}"
GITHUB_BASE_URL="${GITHUB_BASE_URL:-https://github.com}"
VERSION="${ACME233_VERSION:-latest}"
INSTALL_DIR="${ACME233_INSTALL_DIR:-$HOME/.local/bin}"
NO_PATH=0
DRY_RUN=0

usage() {
  cat <<'EOF'
Install acme233 on Linux or macOS.

Usage:
  curl -fsSL https://raw.githubusercontent.com/neko233-com/acme233/main/install.sh | sh
  curl -fsSL https://raw.githubusercontent.com/neko233-com/acme233/main/install.sh | sh -s -- --version v0.0.1

Options:
  --version <version>      Install a release tag. Defaults to latest.
  --install-dir <dir>      Install directory. Defaults to $HOME/.local/bin.
  --no-path                Do not append the install directory to a shell profile.
  --dry-run                Print what would be installed without changing files.
  -h, --help               Show this help.

Environment:
  ACME233_VERSION          Same as --version.
  ACME233_INSTALL_DIR      Same as --install-dir.
  ACME233_REPO             GitHub repository, owner/name. Defaults to neko233-com/acme233.
  GITHUB_BASE_URL          GitHub base URL. Defaults to https://github.com.
  GITHUB_TOKEN             Optional token for authenticated downloads.
EOF
}

while [ $# -gt 0 ]; do
  case "$1" in
    --version)
      VERSION="${2:?missing value for --version}"
      shift 2
      ;;
    --install-dir)
      INSTALL_DIR="${2:?missing value for --install-dir}"
      shift 2
      ;;
    --no-path)
      NO_PATH=1
      shift
      ;;
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown option: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

need() {
  if ! command -v "$1" >/dev/null 2>&1; then
    echo "required command not found: $1" >&2
    exit 1
  fi
}

need curl

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$os" in
  linux*) goos="linux" ;;
  darwin*) goos="darwin" ;;
  *)
    echo "unsupported OS: $os" >&2
    exit 1
    ;;
esac

machine="$(uname -m)"
case "$machine" in
  x86_64|amd64) goarch="amd64" ;;
  arm64|aarch64) goarch="arm64" ;;
  *)
    echo "unsupported architecture: $machine" >&2
    exit 1
    ;;
esac

if [ "$VERSION" != "latest" ] && [ "${VERSION#v}" = "$VERSION" ]; then
  VERSION="v$VERSION"
fi

asset="acme233_${goos}_${goarch}.zip"
if [ "$VERSION" = "latest" ]; then
  download_url="${GITHUB_BASE_URL%/}/$REPO/releases/latest/download/$asset"
else
  download_url="${GITHUB_BASE_URL%/}/$REPO/releases/download/$VERSION/$asset"
fi

auth_header=""
if [ "${GITHUB_TOKEN:-}" != "" ]; then
  auth_header="Authorization: Bearer $GITHUB_TOKEN"
fi

echo "Installing $asset to $INSTALL_DIR"
if [ "$DRY_RUN" = "1" ]; then
  echo "dry run: would download $download_url"
  echo "dry run: would install acme233 into $INSTALL_DIR"
  exit 0
fi

tmpdir="$(mktemp -d 2>/dev/null || mktemp -d -t acme233)"
cleanup() {
  rm -rf "$tmpdir"
}
trap cleanup EXIT INT TERM

archive="$tmpdir/$asset"
if [ "$auth_header" != "" ]; then
  curl -fL -H "$auth_header" -o "$archive" "$download_url"
else
  curl -fL -o "$archive" "$download_url"
fi

if command -v unzip >/dev/null 2>&1; then
  unzip -q "$archive" -d "$tmpdir"
elif command -v python3 >/dev/null 2>&1; then
  python3 - "$archive" "$tmpdir" <<'PY'
import sys
import zipfile

with zipfile.ZipFile(sys.argv[1]) as archive:
    archive.extractall(sys.argv[2])
PY
else
  echo "required command not found: unzip or python3" >&2
  exit 1
fi

binary_path="$(find "$tmpdir" -type f -name acme233 | head -n 1)"
if [ "$binary_path" = "" ]; then
  echo "acme233 binary not found in release asset" >&2
  exit 1
fi

mkdir -p "$INSTALL_DIR"
cp "$binary_path" "$INSTALL_DIR/acme233"
chmod 755 "$INSTALL_DIR/acme233"

case ":$PATH:" in
  *":$INSTALL_DIR:"*) in_path=1 ;;
  *) in_path=0 ;;
esac

if [ "$NO_PATH" = "0" ] && [ "$in_path" = "0" ]; then
  profile="${SHELL:-}"
  case "$profile" in
    */zsh) profile_file="$HOME/.zshrc" ;;
    */bash) profile_file="$HOME/.bashrc" ;;
    *) profile_file="$HOME/.profile" ;;
  esac
  touch "$profile_file"
  if ! grep -F "$INSTALL_DIR" "$profile_file" >/dev/null 2>&1; then
    {
      echo ''
      echo "# acme233"
      echo "export PATH=\"$INSTALL_DIR:\$PATH\""
    } >> "$profile_file"
    echo "Added $INSTALL_DIR to PATH in $profile_file"
  fi
fi

echo "acme233 installed: $INSTALL_DIR/acme233"
echo "Run '$INSTALL_DIR/acme233 version' to verify."
if [ "$in_path" = "0" ]; then
  echo "Open a new terminal or run: export PATH=\"$INSTALL_DIR:\$PATH\""
fi
