#!/usr/bin/env bash
set -euo pipefail

CLUSTER_NAME=${CLUSTER_NAME:-"3fs-csi-kind"}

kind create cluster --name "$CLUSTER_NAME" || true

kubectl create ns 3fs-csi || true
helm upgrade --install 3fs-csi ./deploy/helm/3fs-csi -n 3fs-csi \
  --set clusterId=${CLUSTER_ID:-stage} \
  --set mgmtdAddresses='["RDMA://127.0.0.1:8000"]'

kubectl apply -f deploy/examples/pv-pvc.yaml
kubectl apply -f deploy/examples/two-pods.yaml

echo "Waiting for pods..."
kubectl wait --for=condition=Ready pod/rwx-writer --timeout=120s || true
kubectl wait --for=condition=Ready pod/rwx-reader --timeout=120s || true

echo "Checking shared file..."
kubectl exec rwx-writer -- sh -c 'ls -l /data/shared && cat /data/shared/file.txt || true'
kubectl exec rwx-reader -- sh -c 'ls -l /data/shared && cat /data/shared/file.txt || true'

echo "Done"

