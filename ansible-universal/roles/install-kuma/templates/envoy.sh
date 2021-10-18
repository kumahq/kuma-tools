#! /usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

readonly DOCKER=${DOCKER:-podman}
readonly ENVOY=${ENVOY:-{{envoy.repository}}:{{envoy.version}}}

readonly HERE=$(cd $(dirname $0) && pwd)
readonly ENVOY_BIN="$HERE/.bin/envoy.{{envoy.version}}"

# This is just atrocious. podman ignores `--user=root` which we need to be able
# to read the envoy bootstrap, so avoid the whole mess of containerization by
# copying the envoy binary out of the container. Surprisingly, the binary from
# the Envoy Ubuntu-based image works fine on Fedora  ¯\_(ツ)_/¯
if [ ! -x "$ENVOY_BIN" ]; then
  readonly CONTAINERID="envoy-binary-source-$RANDOM"

  mkdir -p $(dirname "$ENVOY_BIN")

  "$DOCKER" run --detach --security-opt label=disable --name "$CONTAINERID" "$ENVOY" sleep 1d
  "$DOCKER" cp "$CONTAINERID":/usr/local/bin/envoy "$ENVOY_BIN"
  "$DOCKER" kill "$CONTAINERID"
fi

exec "$ENVOY_BIN" "$@"
