#!/usr/bin/env bash
set -euo pipefail
NODE_URL="${NODE_URL:-http://127.0.0.1:30999}"
AUTHOR="${AUTHOR:-alice}"
OUT="${OUT:-./downloads}"
mkdir -p "${OUT}"
sudo nerdctl run --rm \
  -e NODE_URL="$NODE_URL" -e AUTHOR="$AUTHOR" \
  -v "$PWD":/work -w /work \
  cipherlot-clt:v0.0.1 \
  subscribe --author "${AUTHOR}" --node "${NODE_URL}" --out /work/"${OUT}"
