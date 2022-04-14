#!/bin/bash -xe

CLUSTER_ENDPOINT=$1
CERTIFICATE_AUTHORITY_DATA=$2
CLUSTER_NAME=$3

sudo /etc/eks/bootstrap.sh --apiserver-endpoint "${CLUSTER_ENDPOINT}" --b64-cluster-ca "${CERTIFICATE_AUTHORITY_DATA}" "${CLUSTER_NAME}"

