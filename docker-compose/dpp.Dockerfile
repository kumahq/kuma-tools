FROM kumahq/kumactl:1.2.3 AS kumactl

FROM kumahq/kuma-dp:1.2.3 AS kumadp

FROM envoyproxy/envoy-alpine:v1.18.4
RUN addgroup kuma-dp -g 5678
RUN adduser -S -D kuma-dp -g kuma-dp -u 5678
COPY --from=kumactl /usr/bin/kumactl /usr/bin/kumactl
COPY --from=kumadp /usr/bin/kuma-dp /usr/bin/kuma-dp
COPY --from=kumadp /usr/bin/coredns /usr/bin/coredns
RUN apk add iptables
RUN apk add sudo
