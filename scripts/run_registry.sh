#!/bin/bash
set -e

# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

DOCKER=$(which docker)

echo "Ensuring local docker registry is running..."
# if the registry is not running, we won't get a match, so we'll start it
if [ -z "$($DOCKER ps --filter name="^/registry$" --no-trunc -q)" ] ; then
    echo -n "Starting local docker registry... "
    $DOCKER run --restart=always -d -p 127.0.0.1:5000:5000 --name registry registry:latest
    echo "SUCCESS"
else
    echo "Local docker registry is already running"
fi
echo
