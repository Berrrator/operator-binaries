#!/usr/bin/env bash
set -euo pipefail

OWNER="Berrrator"
REPO="operator-binaries"

# --- Detect OS ---
case "$(uname -s)" in
  Linux*)   GOOS=linux;;
  Darwin*)  GOOS=darwin;;
  CYGWIN*|MINGW*|MSYS*) GOOS=windows;;
  *)        echo "Unsupported OS: $(uname -s)"; exit 1;;
esac

# --- Detect Arch ---
case "$(uname -m)" in
  x86_64)        GOARCH=amd64;;
  arm64|aarch64) GOARCH=arm64;;
  *) echo "Unsupported architecture: $(uname -m)"; exit 1;;
esac

# --- Get latest release tag dynamically ---
echo "Fetching latest release version from GitHub..."
LATEST_TAG=$(curl -s https://api.github.com/repos/${OWNER}/${REPO}/releases/latest | grep -Po '"tag_name":\s*"\K[^"]+')

if [ -z "$LATEST_TAG" ]; then
  echo "Failed to fetch the latest release tag."
  exit 1
fi

REPO_URL="https://github.com/${OWNER}/${REPO}/releases/download/${LATEST_TAG}"

# --- Construct binary name for download ---
REMOTE_BINARY="operator_wrapper_${GOOS}_${GOARCH}"
[[ "$GOOS" == "windows" ]] && REMOTE_BINARY="${REMOTE_BINARY}.exe"

# --- Local filename (always the same) ---
LOCAL_BINARY="operator_wrapper"
[[ "$GOOS" == "windows" ]] && LOCAL_BINARY="${LOCAL_BINARY}.exe"

echo "Platform detected: ${GOOS}/${GOARCH}"
echo "Downloading ${REMOTE_BINARY} from release ${LATEST_TAG}"
echo "   ${REPO_URL}/${REMOTE_BINARY}"
echo

# --- Download Binary ---
curl -fL -o "${LOCAL_BINARY}" "${REPO_URL}/${REMOTE_BINARY}"
chmod +x "${LOCAL_BINARY}"

echo "Successfully downloaded ${LOCAL_BINARY} (from ${REMOTE_BINARY}, release ${LATEST_TAG})"
echo
echo "Run it like this:"
echo "  ./${LOCAL_BINARY} -operator-priv=\"<private_key>\" -wrapping-pem=\"<wrapping_pem>\""
