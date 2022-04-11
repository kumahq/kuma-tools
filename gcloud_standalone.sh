#!/usr/bin/env bash
set -ex

MACHINE_TYPE=${MACHINE_TYPE:-n1-standard-16}
NUM_NODES=${NUM_NODES:-2}
CLUSTER_NAME=${CLUSTER_NAME:-kuma-tools-test}
REGION=${REGION:-europe-west1-c}
KUMA_HELM_CHART=${KUMA_HELM_CHART:-kuma/kuma}
KUMA_VALUES=${KUMA_VALUES:-values.yaml}
#
gcloud container clusters create ${CLUSTER_NAME} --num-nodes=${NUM_NODES} --zone ${REGION} --preemptible --machine-type ${MACHINE_TYPE}
gcloud container clusters get-credentials ${CLUSTER_NAME} --region ${REGION}

until kubectl wait -n kube-system --timeout=30s --for condition=Ready --all pods; do sleep 5; done

helm repo add kuma https://kumahq.github.io/charts
helm repo update
helm upgrade -i --create-namespace --namespace kuma-system kuma ${KUMA_HELM_CHART} -f ${KUMA_VALUES}

echo "Waiting for cp pods to be ready"
kubectl wait -n kuma-system --timeout=30s --for condition=Ready --all pods

helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
echo '
alertmanager:
  enabled: false
nodeExporter:
  enabled: false
kubeStateMetrics:
  enabled: false
kubeProxy:
  enabled: false
kubeScheduler:
  enabled: false
kubeEtcd:
  enabled: false
coreDns:
  enabled: false
kubeControllerManager:
  enabled: false
kubelet:
  enabled: false
kubeApiServer:
  enabled: false
grafana:
  service:
    type: LoadBalancer
prometheusOperator:
  namespaces:
    releaseNamespace: true
    additional:
      - kuma-system
#  prometheusSpec:
#    additionalScrapeConfigs:
#        - job_name: 'kuma-dataplanes'
#          scrape_interval: "5s"
#          relabel_configs:
#          - source_labels:
#            - __meta_kuma_mesh
#            regex: "(.*)"
#            target_label: mesh
#          - source_labels:
#            - __meta_kuma_dataplane
#            regex: "(.*)"
#            target_label: dataplane
#          - source_labels:
#            - __meta_kuma_service
#            regex: "(.*)"
#            target_label: service
#          - action: labelmap
#            regex: __meta_kuma_label_(.+)
#          kuma_sd_configs:
#          - server: "http://kuma-control-plane.kuma-system.svc:5676"
' | helm upgrade -i prom-stack prometheus-community/kube-prometheus-stack --create-namespace --namespace observability -f -

echo "Waiting for prometheus"
kubectl wait -n observability --timeout=30s --for condition=Ready --all pods

echo "Add service monitor for CP"
echo '
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: kuma-cp
  namespace: observability
  labels:
    release: prom-stack
spec:
  endpoints:
    - path: /metrics
      port: diagnostics
      scheme: http
  namespaceSelector:
    matchNames:
      - kuma-system
  selector:
    matchLabels:
      app.kubernetes.io/instance: kuma
' | kubectl apply -f -

echo "Add service for CP GUI"
echo '
apiVersion: v1
kind: Service
metadata:
  name: kuma-control-plane-gui
  namespace: kuma-system
spec:
  ports:
  - name: http-api-server
    port: 80
    protocol: TCP
    targetPort: 5681
  selector:
    app: kuma-control-plane
    app.kubernetes.io/instance: kuma
    app.kubernetes.io/name: kuma
  sessionAffinity: None
  type: LoadBalancer
' | kubectl apply -f -

until [[ `kubectl get svc -n observability prom-stack-grafana -ojson | jq -r '.status.loadBalancer.ingress[0].ip'` != "null" ]]; do sleep 5; done
until [[ `kubectl get svc -n kuma-system kuma-control-plane-gui -ojson | jq -r '.status.loadBalancer.ingress[0].ip'` != "null" ]]; do sleep 5; done
echo "
grafana ui: http://`kubectl get svc -n observability prom-stack-grafana -ojson | jq -r '.status.loadBalancer.ingress[0].ip'`
username: admin, password: prom-operator
Kuma gui: http://`kubectl get svc -n kuma-system kuma-control-plane-gui -ojson | jq -r '.status.loadBalancer.ingress[0].ip'`/gui
"
