#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
pgrep -f buildkitd >/dev/null || sudo buildkitd --oci-worker=false --containerd-worker=true &>/dev/null & sleep 2
cd "${ROOT_DIR}/cipherlot-client"
sudo nerdctl build -t cipherlot-clt:v0.0.1 .
