#!/bin/bash
# Copyright 2023 Hedgehog
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

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
