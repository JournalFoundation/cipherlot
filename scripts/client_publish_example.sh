#!/usr/bin/env bash
set -euo pipefail
NODE_URL="${NODE_URL:-http://127.0.0.1:30999}"
AUTHOR="${AUTHOR:-alice}"
FILE="${1:-/etc/hosts}"   # demo file; override with a path

# run client container, mount host cwd for convenience
sudo nerdctl run --rm \
  -e NODE_URL="$NODE_URL" -e AUTHOR="$AUTHOR" \
  -v "$PWD":/work -w /work \
  cipherlot-clt:v0.0.1 \
  publish --author "${AUTHOR}" --node "${NODE_URL}" "${FILE}"
