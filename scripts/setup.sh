#!/bin/sh

set -e

[ -z "${DOCKER_TAG}" ] && echo "DOCKER_TAG is not set" && exit 1

if ! [ -x "$(command -v k3d)" ]; then
  echo 'Error: k3d is not installed. Please visit https://k3d.io for further details.' >&2
  exit 1
fi

retry_logs() {
  n=0
  until [ "$n" -ge $MAX_RETRIES ]; do
    kubectl logs "$1" && break
    n=$((n + 1))
    sleep 5
  done
}

MAX_RETRIES=10
CLUSTER=investigate-go-configmap
NAMESPACE=investigate

# Delete old cluster before attempting to run the test
if (k3d cluster list | grep -q ${CLUSTER}); then
  echo "Deleting old cluster ${CLUSTER}"
  k3d cluster delete $CLUSTER
fi

# Create new kubernetes cluster
k3d cluster create ${CLUSTER}
sleep 2

for img in ${DOCKER_TAG}; do
  echo "Importing image $img built at $(docker inspect -f '{{ .Created }}' $img)"
  k3d images import "$img" -c ${CLUSTER}
done

# Create namespace
kubectl create -f "k8s/namespace.json"
kubectl create -f "k8s/signer-namespace.json"
kubectl config set-context --current --namespace ${NAMESPACE}

# Get dynamically assigned IP of nfs server
kubectl apply -f k8s/investigate-go/pod.yaml

# Turn on/off for debugging
kubectl apply -f k8s/debug/pod.yaml

retry_logs investigate-go-configmap
kubectl logs investigate-go-configmap

# Signer namespace
retry_logs debug
kubectl logs debug -n signer
retry_logs signer-pod
kubectl logs signer-pod -n signer