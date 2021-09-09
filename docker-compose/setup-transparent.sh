#!/bin/bash

kumactl install transparent-proxy --skip-resolv-conf --redirect-dns --kuma-dp-user kuma-dp --verbose --redirect-inbound --redirect-dns-upstream-target-chain DOCKER_OUTPUT
ADDRESS=`ip route list | tail -1 | awk '{ print $7 }'`
if [[ -z "${DP_FILE}" ]]; then
cat <<-EOF > /tmp/dp.yaml
type: Dataplane
mesh: default
name: {{DP_NAME}}
networking:
  address: {{ADDRESS}}
  inbound:
    - port: ${PORT}
      servicePort: ${SERVICE_PORT}
      tags:
        kuma.io/service: ${SERVICE_NAME}
  transparentProxying:
    redirectPortInbound: 15006
    redirectPortOutbound: 15001

EOF
else
  cp ${DP_FILE} /tmp/dp.yaml
fi
cat /tmp/dp.yaml
sudo -u kuma-dp /usr/bin/kuma-dp run --cp-address=https://${CP_HOSTNAME}:5678 --log-level=debug --dataplane-var=ADDRESS=${ADDRESS} --dataplane-var=DP_NAME="${SERVICE_NAME}-${HOSTNAME}" --dataplane-file=/tmp/dp.yaml --dns-coredns-config-template-path=${COREDNS_CONF}
