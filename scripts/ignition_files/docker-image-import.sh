#!/bin/bash
set -e

# get a list of all docker images that we want to import
cd /opt/docker-images
IMAGES=$(ls -1)

# now simply import them into the k3s containerd images
# reusing k3s here has the advantage that it is interacting
# with the right container runtime and namespace, etc.pp.
for i in $IMAGES; do
  /opt/bin/k3s ctr images import $i
done

# let the systemd unit know that we don't need to start this again
touch /opt/docker-images-imported
