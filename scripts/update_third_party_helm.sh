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


# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# preparing all third_party helm/docker installations
echo "Preparing all 3rd party products for installation..."
echo "Preparing rsyslog for installation..."
( cd $SCRIPT_DIR/../third_party/helm/rsyslog-server && docker build -t registry.local:31000/githedgehog/rsyslog:0.1.0 . )
skopeo copy --dest-tls-verify=false docker-daemon:registry.local:31000/githedgehog/rsyslog:0.1.0 docker://registry.local:31000/githedgehog/rsyslog:0.1.0
helm package $SCRIPT_DIR/../third_party/helm/rsyslog-server/charts/rsyslog --version 0.1.2 --app-version 0.1.0 -d $SCRIPT_DIR/../third_party
helm push --insecure-skip-tls-verify $SCRIPT_DIR/../third_party/rsyslog-0.1.2.tgz oci://registry.local:31000/githedgehog/helm-charts
echo

echo "Preparing ntp/chrony for installation..."
docker pull cturra/ntp:latest
docker tag cturra/ntp:latest registry.local:31000/cturra/ntp:latest
skopeo copy --dest-tls-verify=false docker-daemon:registry.local:31000/cturra/ntp:latest docker://registry.local:31000/cturra/ntp:latest
docker push registry.local:31000/cturra/ntp:latest
helm package $SCRIPT_DIR/../third_party/helm/ntp --version 0.0.2 -d $SCRIPT_DIR/../third_party
helm push --insecure-skip-tls-verify $SCRIPT_DIR/../third_party/ntp-0.0.2.tgz oci://registry.local:31000/githedgehog/helm-charts
echo
