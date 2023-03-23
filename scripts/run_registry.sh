#!/bin/bash
set -e

# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# ensure repo certs have been created before already
mkdir -p ${SCRIPT_DIR}/../dev/oci
OCI_CERT_DIR=$( cd -- "${SCRIPT_DIR}/../dev/oci" &> /dev/null && pwd )

if [ ! -f $OCI_CERT_DIR/server-cert.pem -o ! -f $OCI_CERT_DIR/server-key.pem ] ; then
    $SCRIPT_DIR/init_repo_certs.sh
fi

DOCKER=$(which docker)

echo "Ensuring local docker registry is running..."
# if the registry is not running, we won't get a match, so we'll start it
# NOTE: we're disabling http2 as that seems to be causing issues with containerd sometimes
if [ -z "$($DOCKER ps --filter name="^/registry$" --no-trunc -q)" ] ; then
    echo -n "Starting local docker registry... "
    $DOCKER run -d \
      --restart=always \
      -p 127.0.0.1:5000:5000 \
      -v $OCI_CERT_DIR:/certs \
      -e REGISTRY_HTTP_ADDR=0.0.0.0:5000 \
      -e REGISTRY_HTTP_HOST=https://registry.local:5000 \
      -e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/server-cert.pem \
      -e REGISTRY_HTTP_TLS_KEY=/certs/server-key.pem \
      -e REGISTRY_HTTP_HTTP2_DISABLED=true \
      --name registry \
      registry:latest
    echo "SUCCESS"
else
    echo "Local docker registry is already running"
fi
echo
