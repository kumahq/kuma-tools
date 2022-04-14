#!/bin/bash

TEST_ID=$1
DATAPLANE_ID=$2
CP_IP=$3

echo "
contexts:
- controlPlane: local
  name: local
controlPlanes:
- coordinates:
    apiServer:
      url: http://${CP_ID}:5681
  name: local
currentContext: local
" > /home/ubuntu/kumaconfig
chown ubuntu:ubuntu /home/ubuntu/kumaconfig

echo "type: Dataplane
mesh: default
name: backend-${DATAPLANE_ID}
networking:
  address: 127.0.0.1
  inbound:
  - port: 21011
    servicePort: 21012
    tags:
      kuma.io/service: backend-${DATAPLANE_ID}
      kuma.io/protocol: http
" > /home/ubuntu/dp.yaml
chown ubuntu:ubuntu /home/ubuntu/dp.yaml

aws s3 cp s3://kuma-benchmarking/tests/${TEST_ID}/token /home/ubuntu/token
mkdir /home/ubuntu/kuma_pkg
aws s3 cp s3://kuma-benchmarking/tests/${TEST_ID}/kuma.tgz /home/ubuntu/kuma_pkg/kuma.tgz
cd /home/ubuntu/kuma_pkg
tar -xzpvf ./kuma.tgz
./kuma-*/kuma-dp run   --cp-address=https://${CP_IP}:5678 --dataplane-token-file=/home/ubuntu/token --dataplane-file=/home/ubuntu/dp.yaml
