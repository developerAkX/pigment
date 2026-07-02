#!/usr/bin/env bash
set -euo pipefail

# pigment installer for macOS and Linux

REPO="developerAkX/pigment"

# Config via environment:
#   PIGMENT_VERSION      pin a specific version (e.g. 0.1.0); default = latest
#   PIGMENT_INSTALL_DIR  override install directory
#   PIGMENT_SKILLS_AGENT agent(s) to install skills for (default: opencode; '*' = all)
#   PIGMENT_NO_SKILLS=1  skip `npx skills add` skill installation
SKILLS_AGENT="${PIGMENT_SKILLS_AGENT:-opencode}"

main() {
  echo "Installing pigment..."
  echo ""

  detect_platform
  fetch_latest_version
  download_and_verify
  install_binary
  install_skills
  post_install
}

detect_platform() {
  OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
  ARCH="$(uname -m)"

  case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
  esac

  case "$OS" in
    darwin|linux) ;;
    *) echo "Unsupported OS: $OS (use install.ps1 for Windows)"; exit 1 ;;
  esac

  echo "Platform: ${OS}/${ARCH}"
}

fetch_latest_version() {
  if [ -n "${PIGMENT_VERSION:-}" ]; then
    VERSION="${PIGMENT_VERSION#v}"
    echo "Requested version: ${VERSION}"
    return
  fi

  VERSION=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" \
    | grep '"tag_name"' | head -1 | sed 's/.*"v\(.*\)".*/\1/')

  if [ -z "$VERSION" ]; then
    echo "Failed to determine latest version."
    exit 1
  fi

  echo "Latest version: ${VERSION}"
}

download_and_verify() {
  ASSET="pigment_${VERSION}_${OS}_${ARCH}.tar.gz"
  BASE_URL="https://github.com/${REPO}/releases/download/v${VERSION}"

  TMPDIR="$(mktemp -d)"
  trap 'rm -rf "$TMPDIR"' EXIT

  echo "Downloading ${ASSET}..."
  curl -fsSL "${BASE_URL}/${ASSET}" -o "${TMPDIR}/${ASSET}"
  curl -fsSL "${BASE_URL}/checksums.txt" -o "${TMPDIR}/checksums.txt"

  echo "Verifying checksum..."
  EXPECTED=$(grep "${ASSET}" "${TMPDIR}/checksums.txt" | awk '{print $1}')
  if [ -z "$EXPECTED" ]; then
    echo "Checksum not found for ${ASSET}"
    exit 1
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL=$(sha256sum "${TMPDIR}/${ASSET}" | awk '{print $1}')
  else
    ACTUAL=$(shasum -a 256 "${TMPDIR}/${ASSET}" | awk '{print $1}')
  fi

  if [ "$ACTUAL" != "$EXPECTED" ]; then
    echo "Checksum mismatch!"
    echo "  Expected: ${EXPECTED}"
    echo "  Actual:   ${ACTUAL}"
    exit 1
  fi
  echo "Checksum OK."

  echo "Extracting..."
  tar -xzf "${TMPDIR}/${ASSET}" -C "${TMPDIR}"
}

install_binary() {
  INSTALL_DIR="${PIGMENT_INSTALL_DIR:-/usr/local/bin}"
  if [ ! -d "$INSTALL_DIR" ] || [ ! -w "$INSTALL_DIR" ] 2>/dev/null; then
    INSTALL_DIR="${HOME}/.local/bin"
    mkdir -p "$INSTALL_DIR"
  fi

  cp "${TMPDIR}/pigment" "${INSTALL_DIR}/pigment"
  chmod +x "${INSTALL_DIR}/pigment"
  BIN_PATH="${INSTALL_DIR}/pigment"
  echo "Installed binary to ${BIN_PATH}"

  # Check PATH
  case ":${PATH}:" in
    *":${INSTALL_DIR}:"*) ;;
    *) echo "NOTE: ${INSTALL_DIR} is not on your PATH — add it to use 'pigment' directly." ;;
  esac
}

# Install the agent skills into your agent's skill directory via the
# skills.sh registry (`npx skills add`). Falls back to the binary's own
# embedded installer if Node/npx is unavailable. Set PIGMENT_NO_SKILLS=1
# to skip entirely.
install_skills() {
  echo ""
  if [ "${PIGMENT_NO_SKILLS:-0}" = "1" ]; then
    echo "Skipping skill installation (PIGMENT_NO_SKILLS=1)."
    return
  fi

  echo "Installing agent skills (agent: ${SKILLS_AGENT})..."
  if command -v npx >/dev/null 2>&1; then
    if npx -y skills@latest add "${REPO}" \
        --skill '*' --agent "${SKILLS_AGENT}" --global --yes; then
      echo "Skills installed via 'npx skills add'."
      return
    fi
    echo "WARN: 'npx skills add' failed; falling back to the embedded installer."
  else
    echo "npx/Node not found; using the embedded skill installer."
  fi

  # Fallback: use the binary's built-in installer (opencode target).
  if [ -n "${BIN_PATH:-}" ] && [ -x "${BIN_PATH}" ]; then
    "${BIN_PATH}" skill install --force || \
      echo "WARN: embedded skill install failed; run 'pigment skill install' manually."
  fi
}

post_install() {
  echo ""
  echo "Done! Next steps:"
  echo "  1. Authenticate your ChatGPT subscription:  codex login"
  echo "  2. Verify everything is ready:              pigment doctor"
  echo "  3. Generate your first image:               pigment gen \"a red bicycle\""
  echo ""
  echo "Skills installed for '${SKILLS_AGENT}'. Re-run any time with:"
  echo "  npx skills add ${REPO} --agent <agent> --global --yes"
  echo ""

  # Best-effort readiness check.
  if [ -n "${BIN_PATH:-}" ] && [ -x "${BIN_PATH}" ]; then
    echo "Running 'pigment doctor':"
    "${BIN_PATH}" doctor || true
  fi
}

main "$@"
