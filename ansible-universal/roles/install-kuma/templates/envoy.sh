#! /usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

readonly DOCKER=${DOCKER:-podman}
readonly ENVOY=${ENVOY:-{{envoy.repository}}:{{envoy.version}}}

exec $DOCKER run --network host $ENVOY "$@"

