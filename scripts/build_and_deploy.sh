#!/usr/bin/env bash
set -euo pipefail
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NODE_IP="${NODE_IP:-192.168.29.10}"   # default for Centroid
NODEPORT="${NODEPORT:-30999}"

# preflight: verify NodePort is free on this host
if ss -tln sport = :${NODEPORT} | grep -q ":${NODEPORT}"; then
  echo "ERROR: TCP ${NODEPORT} already in use. Choose a free NodePort and re-run:"
  echo "  NODEPORT=31xxx bash ${ROOT_DIR}/scripts/build_and_deploy.sh"
  exit 1
fi

# ensure buildkitd for nerdctl builds
pgrep -f buildkitd >/dev/null || sudo buildkitd --oci-worker=false --containerd-worker=true &>/dev/null & sleep 2

# build node image
echo ">> Building cipherlot-node image"
cd "${ROOT_DIR}/cipherlot-node"
sudo nerdctl build -t cipherlot-node:v0.0.1 .

# load into k8s.io namespace for kubelet
echo ">> Loading image into containerd k8s.io namespace"
sudo nerdctl save -o /tmp/cipherlot-node.tar cipherlot-node:v0.0.1
sudo nerdctl --namespace k8s.io load -i /tmp/cipherlot-node.tar
rm -f /tmp/cipherlot-node.tar

# deploy to Kubernetes
echo ">> Applying Kubernetes manifest"
cd "${ROOT_DIR}"
kubectl apply -f cipherlot-node.yaml

echo ">> Waiting for pod to be Ready"
kubectl -n cipherlot rollout status deploy/cipherlot-node --timeout=90s

echo ">> Node health check"
curl -sf "http://127.0.0.1:${NODEPORT}/health" || curl -sf "http://$NODE_IP:${NODEPORT}/health"
echo
