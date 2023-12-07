# Universal on Kubernetes

This is not a usuall setup that a user deploy.
However, it's useful for testing universal standalone without spawning multiple VMs etc.

## Create namespaces

kubectl create namespace kong-mesh-system
kubectl create namespace postgres

## Install Postgres

helm install postgres -n postgres \
  --set global.postgresql.auth.username=kongmesh \
  --set global.postgresql.auth.database=kongmesh \
  oci://registry-1.docker.io/bitnamicharts/postgresql

## Install Kong Mesh

### Prepare Postgres credential for Kong Mesh

export POSTGRES_USER_DB=$(echo -n "kongmesh" | base64)
export POSTGRES_PASSWORD=$(kubectl get secret --namespace postgres postgres-postgresql -o jsonpath="{.data.password}")
export POSTGRES_HOST=$(echo -n "postgres-postgresql.postgres.svc.cluster.local" | base64)
echo "
apiVersion: v1
kind: Secret
metadata:
  name: db-creds
  namespace: kong-mesh-system
type: Opaque
data:
  POSTGRES_DB: $POSTGRES_USER_DB
  POSTGRES_HOST_RW: $POSTGRES_HOST
  POSTGRES_USER: $POSTGRES_USER_DB
  POSTGRES_PASSWORD: $POSTGRES_PASSWORD
" | kubectl apply -f -

### Generate certs

We need to generate certs and set them manually. Normally HELM chart would handle this.

kumactl generate tls-certificate --type=server --hostname=kong-mesh-control-plane.kong-mesh-system
kubectl create secret -n kong-mesh-system generic tls-general --from-file=tls.crt=cert.pem --from-file=tls.key=key.pem --from-file=ca.crt=cert.pem

### Install

echo "
kuma:
  controlPlane:
    environment: universal
    mode: standalone
    secrets:
      postgresDb:
        Secret: db-creds
        Key: POSTGRES_DB
        Env: KUMA_STORE_POSTGRES_DB_NAME
      postgresHost:
        Secret: db-creds
        Key: POSTGRES_HOST_RW
        Env: KUMA_STORE_POSTGRES_HOST
      postgrestUser:
        Secret: db-creds
        Key: POSTGRES_USER
        Env: KUMA_STORE_POSTGRES_USER
      postgresPassword:
        Secret: db-creds
        Key: POSTGRES_PASSWORD
        Env: KUMA_STORE_POSTGRES_PASSWORD
    envVars:
      KUMA_API_SERVER_AUTHN_LOCALHOST_IS_ADMIN: true
      KUMA_GENERAL_TLS_CERT_FILE: /var/run/secrets/kuma.io/certs/tls.crt
      KUMA_GENERAL_TLS_KEY_FILE: /var/run/secrets/kuma.io/certs/tls.key
    extraSecrets:
      extraConfig:
        name: tls-general
        mountPath: /var/run/secrets/kuma.io/certs
        readOnly: true
" > values.yaml
helm install kong-mesh -f values.yaml --skip-crds --namespace kong-mesh-system kong-mesh/kong-mesh

## Configure kumactl

kubectl port-forward -n kong-mesh-system svc/kong-mesh-control-plane 5681 --address 0.0.0.0
export ADMIN_TOKEN=$(curl localhost:5681/global-secrets/admin-user-token | jq -r '.data' | base64 -d)

kumactl config control-planes add \
  --address http://localhost:5681 \
  --headers "authorization=Bearer $ADMIN_TOKEN" \
  --name "my-cp" \
  --overwrite

## Lock RBAC

Lock RBAC so only admin can apply stuff

echo "
type: AccessRoleBinding
name: default
subjects:
- type: Group
  name: mesh-system:admin
roles:
- admin
" | kumactl apply -f -

edit values.yaml and change `KUMA_API_SERVER_AUTHN_LOCALHOST_IS_ADMIN` to `false`
helm upgrade kong-mesh -f values.yaml --skip-crds --namespace kong-mesh-system kong-mesh/kong-mesh

## Deploy data plane proxy

It deploys just a sidecar without any actuall application. 
It's a simplified example. Normally Kubernetes would inject it and handle authentication for a user.

### Generate data plane proxy token

export DP_TOKEN=$(kumactl generate dataplane-token --valid-for 24h --tag kuma.io/service=sample | base64)
echo "
apiVersion: v1
kind: Secret
metadata:
  name: dp-token
  namespace: kong-mesh-system
type: Opaque
data:
  DP_TOKEN: $DP_TOKEN
" | kubectl apply -f -

### Deploy data plane proxy

echo "
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: sample-dp
  namespace: kong-mesh-system
spec:
  replicas: 1
  strategy:
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  selector:
    matchLabels:
      app: sample-dp
  template:
    metadata:
      labels: 
        app: sample-dp
    spec:
      containers:
      - args:
        - run
        - --log-level=info
        - --concurrency=2
        env:
        - name: INSTANCE_IP
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: status.podIP
        - name: KUMA_CONTROL_PLANE_CA_CERT
          valueFrom:
            secretKeyRef:
              key: ca.crt
              name: tls-general
        - name: KUMA_CONTROL_PLANE_URL
          value: https://kong-mesh-control-plane.kong-mesh-system:5678
        - name: KUMA_DATAPLANE_RUNTIME_RESOURCE
          value: |
            type: Dataplane
            mesh: default
            name: sample-01
            networking:
              address: \$(INSTANCE_IP)
              inbound:
              - port: 8080
                servicePort: 80
                tags:
                  kuma.io/service: sample
        - name: KUMA_DATAPLANE_DRAIN_TIME
          value: 10s
        - name: KUMA_DATAPLANE_RUNTIME_TOKEN
          valueFrom:
            secretKeyRef:
              key: DP_TOKEN
              name: dp-token
        - name: KUMA_DNS_ENABLED
          value: 'false'
        image: kong/kuma-dp:2.5.0
        imagePullPolicy: IfNotPresent
        name: kuma-sidecar
        resources:
          limits:
            memory: 512Mi
          requests:
            cpu: 500m
            memory: 512Mi
        securityContext:
          readOnlyRootFilesystem: true
          runAsGroup: 5678
          runAsUser: 5678
        volumeMounts:
        - mountPath: /tmp
          name: kuma-sidecar-tmp
      volumes:
      - emptyDir: {}
        name: kuma-sidecar-tmp
" | kubectl apply -f -

## Use it

Control plane API is on 5681. Expose it via Ingress/Gateway of your choice just like any other app.