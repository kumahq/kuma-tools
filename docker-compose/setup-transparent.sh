#!/bin/bash

kumactl install transparent-proxy --skip-resolv-conf --redirect-dns --kuma-dp-user kuma-dp --verbose --redirect-inbound --redirect-dns-upstream-target-chain DOCKER_OUTPUT
ADDRESS=`ip route list | tail -1 | awk '{ print $7 }'`
cat <<-EOF > /tmp/dp.yaml
type: Dataplane
mesh: default
name: ${SERVICE_NAME}-${HOSTNAME}
networking:
  address: ${ADDRESS}
  inbound:
    - port: ${PORT}
      servicePort: ${SERVICE_PORT}
      tags:
        kuma.io/service: ${SERVICE_NAME}
  transparentProxying:
    redirectPortInbound: 15006
    redirectPortOutbound: 15001

EOF
cat /tmp/dp.yaml
sudo -u kuma-dp /usr/bin/kuma-dp run --cp-address=https://${CP_HOSTNAME}:5678 --log-level=debug --dataplane-file=/tmp/dp.yaml
