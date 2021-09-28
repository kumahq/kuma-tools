#! /usr/bin/env bash

set -o errexit
set -o pipefail
set -o nounset

readonly SRCROOT="$(cd $(dirname $0)/.. && pwd)"
readonly ENVROOT="${SRCROOT}/.venv"
readonly PYTHON="${PYTHON:-python}"


if [ -d "${ENVROOT}" ]; then
        source "${ENVROOT}/bin/activate"
else
        $PYTHON -m venv "${ENVROOT}"
        source "${ENVROOT}/bin/activate"

        pip install -U pip
        pip install -U ansible paramiko
fi

# Activating the virtualenv prepends the virtualenv bin directory to $PATH, so
# now we can exec the program name out of the virtualenv.
exec $(basename $0) "$@"
