#!/usr/bin/env bash
# Installs unbound-paws (the paws CLI and Claude Code hook) into ~/.local/bin:
#
#   curl -fsSL https://paws.getunbound.ai/install.sh | bash          # latest release
#
# The site redirects to the copy attached to the latest GitHub release.
#   curl -fsSL https://paws.getunbound.ai/install.sh | bash -s 0.3.1 # a given version
#
# It downloads the release archive for this OS and CPU from GitHub, checks it against the
# release's checksums.txt, installs the binary as ~/.local/bin/unbound-paws and, if that folder
# isn't on PATH, adds it in your shell's rc file. It never touches Claude Code's settings: run
# `unbound-paws init` (this repo) or `unbound-paws init --global` (every repo) yourself.
#
# PAWS_INSTALL_DIR  install somewhere else (PATH is then left to you)
# PAWS_DOWNLOAD_BASE  a mirror holding the same release assets (needs a version argument)
# PAWS_NO_MODIFY_PATH=1  never edit a shell rc file
set -euo pipefail

VERSION="${1:-}"
REPO="websentry-ai/paws"
DEFAULT_DIR="$HOME/.local/bin"
INSTALL_DIR="${PAWS_INSTALL_DIR:-$DEFAULT_DIR}"

fail() { echo "unbound-paws install: $*" >&2; exit 1; }

if [ -n "$VERSION" ] && ! [[ "$VERSION" =~ ^v?[0-9]+\.[0-9]+\.[0-9]+(-[0-9A-Za-z.]+)?$ ]]; then
  fail "usage: install.sh [VERSION]  (e.g. 0.3.1)"
fi
VERSION="${VERSION#v}"

# Everything goes under $HOME; under sudo that's root's home, and the command wouldn't be on
# your PATH. Plain root (containers, CI) is fine.
if [ "$(id -u)" -eq 0 ] && [ -n "${SUDO_USER:-}" ] && [ "$SUDO_USER" != "root" ]; then
  fail "don't run this with sudo; it installs into your home folder: curl -fsSL https://paws.getunbound.ai/install.sh | bash"
fi

case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) fail "$(uname -s) isn't supported by this script; on Windows use: npm i -g unbound-paws" ;;
esac
case "$(uname -m)" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
  *) fail "no build for CPU $(uname -m)" ;;
esac
# An x86 shell under Rosetta on Apple silicon still wants the arm64 build.
if [ "$os" = darwin ] && [ "$arch" = amd64 ] && [ "$(sysctl -n sysctl.proc_translated 2>/dev/null || true)" = 1 ]; then
  arch=arm64
fi

if command -v curl >/dev/null 2>&1; then
  fetch() { curl -fsSL -o "$2" "$1"; }
  latest() { curl -fsSLI -o /dev/null -w '%{url_effective}' "https://github.com/$REPO/releases/latest"; }
elif command -v wget >/dev/null 2>&1; then
  fetch() { wget -q -O "$2" "$1"; }
  latest() { wget -q --max-redirect=5 -S -O /dev/null "https://github.com/$REPO/releases/latest" 2>&1 | sed -n 's/^ *Location: *//p' | tail -1; }
else
  fail "needs curl or wget"
fi
if command -v sha256sum >/dev/null 2>&1; then
  sha256() { sha256sum "$1" | cut -d' ' -f1; }
elif command -v shasum >/dev/null 2>&1; then
  sha256() { shasum -a 256 "$1" | cut -d' ' -f1; }
else
  fail "needs sha256sum or shasum to check the download"
fi

if [ -z "$VERSION" ]; then
  [ -z "${PAWS_DOWNLOAD_BASE:-}" ] || fail "PAWS_DOWNLOAD_BASE needs a version argument"
  tag="$(latest)"
  tag="${tag%/}"
  VERSION="${tag##*/v}"
  [[ "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+ ]] || fail "couldn't find the latest release (got '$tag')"
fi
BASE="${PAWS_DOWNLOAD_BASE:-https://github.com/$REPO/releases/download/v$VERSION}"
asset="paws_${VERSION}_${os}_${arch}.tar.gz"

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
echo "Downloading unbound-paws $VERSION ($os/$arch)..."
fetch "$BASE/$asset" "$tmp/$asset" || fail "download failed: $BASE/$asset"
fetch "$BASE/checksums.txt" "$tmp/checksums.txt" || fail "download failed: $BASE/checksums.txt"
want="$(awk -v a="$asset" '$2 == a { print $1 }' "$tmp/checksums.txt")"
[ -n "$want" ] || fail "$asset isn't listed in checksums.txt"
got="$(sha256 "$tmp/$asset")"
[ "$got" = "$want" ] || fail "checksum mismatch for $asset: want $want, got $got"

tar -xzf "$tmp/$asset" -C "$tmp" paws
mkdir -p "$INSTALL_DIR"
# Rename into place: a hook running the old binary keeps its copy; the next call gets the new one.
install -m 0755 "$tmp/paws" "$INSTALL_DIR/.unbound-paws.$$"
mv -f "$INSTALL_DIR/.unbound-paws.$$" "$INSTALL_DIR/unbound-paws"
echo "Installed $("$INSTALL_DIR/unbound-paws" version) at $INSTALL_DIR/unbound-paws"

# `unbound-paws init` run before the folder was on PATH copied the binary here for the hook; keep
# that copy current too (only your own regular file, never through a link).
# Like the npm install: only a newer release (or a copy that no longer runs) replaces it, never an
# older pinned one, and a source build (a version with "dev" in it) is left alone.
newer() { # is $1 (x.y.z[-pre]) newer than $2?
  awk -v a="$1" -v b="$2" 'BEGIN {
    split(a, pa, "-"); split(b, pb, "-"); split(pa[1], x, "."); split(pb[1], y, ".")
    for (i = 1; i <= 3; i++) if (x[i] + 0 != y[i] + 0) exit !(x[i] + 0 > y[i] + 0)
    exit !(pa[2] == "" && pb[2] != "")
  }'
}
hook_copy="${PAWS_HOME:-$HOME/.paws}/bin/paws"
if [ -f "$hook_copy" ] && [ ! -L "$hook_copy" ] && [ -O "$hook_copy" ]; then
  current="$("$hook_copy" version 2>/dev/null || true)"
  current="${current#paws }"
  if [ -z "$current" ] || { [[ "$current" != *dev* ]] && newer "$VERSION" "$current"; }; then
    install -m 0755 "$tmp/paws" "$hook_copy.$$"
    mv -f "$hook_copy.$$" "$hook_copy"
    echo "Updated the hook's copy at $hook_copy"
  fi
fi

case ":$PATH:" in
  *":$INSTALL_DIR:"*) on_path=1 ;;
  *) on_path=0 ;;
esac
if [ "$on_path" = 0 ]; then
  if [ "$INSTALL_DIR" = "$DEFAULT_DIR" ] && [ "${PAWS_NO_MODIFY_PATH:-}" != 1 ]; then
    # shellcheck disable=SC2016  # the line goes into the rc file as written, $HOME and all
    case "$(basename "${SHELL:-}")" in
      zsh) rc="$HOME/.zshrc"; line='export PATH="$HOME/.local/bin:$PATH"' ;;
      bash) if [ "$os" = darwin ]; then rc="$HOME/.bash_profile"; else rc="$HOME/.bashrc"; fi; line='export PATH="$HOME/.local/bin:$PATH"' ;;
      fish) rc="$HOME/.config/fish/config.fish"; line='fish_add_path "$HOME/.local/bin"' ;;
      *) rc="$HOME/.profile"; line='export PATH="$HOME/.local/bin:$PATH"' ;;
    esac
    mkdir -p "$(dirname "$rc")"
    if ! grep -qsF "$line" "$rc"; then
      printf '\n# Added by the unbound-paws installer\n%s\n' "$line" >>"$rc"
      echo "Added ~/.local/bin to your PATH in $rc. Open a new terminal to use unbound-paws."
    fi
  else
    echo "$INSTALL_DIR isn't on your PATH; add it to run unbound-paws by name."
  fi
fi

echo
echo "Next, turn paws on:"
echo "  unbound-paws init --global   # every project"
echo "  unbound-paws init            # just this repo"
