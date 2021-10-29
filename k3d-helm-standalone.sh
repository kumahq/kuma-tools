#!/bin/sh

# A simple script to start and wait for k3d cluster to be started with a standalone control-plane installed with the latest version of the kuma helm chart

k3d cluster create kuma-demo --k3s-server-arg '--no-deploy=traefik'   --k3s-server-arg '--disable=metrics-server' --wait --timeout 60s --no-lb

kubectl config use-context k3d-kuma-demo
until kubectl wait -n kube-system --timeout=30s --for condition=Ready --all pods; do sleep 5; done

helm repo add kuma https://kumahq.github.io/charts
helm repo update
helm install --create-namespace --namespace kuma-system kuma kuma/kuma
echo "Waiting for cp pods to be ready"
kubectl wait -n kuma-system --timeout=30s --for condition=Ready --all pods
