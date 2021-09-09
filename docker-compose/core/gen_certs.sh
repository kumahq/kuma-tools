#!/bin/bash

rm -f key.pem cert.pem
docker run -v `pwd`:/builder -w /builder kumahq/kumactl:1.3.0 kumactl generate tls-certificate --type=server --cp-hostname=kuma-control-plane
