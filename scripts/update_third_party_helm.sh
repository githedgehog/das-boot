#!/bin/bash

# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# preparing all third_party helm/docker installations
echo "Preparing all 3rd party products for installation..."
echo "Preparing rsyslog for installation..."
( cd $SCRIPT_DIR/../third_party/helm/rsyslog-server && docker build -t registry.local:31000/githedgehog/rsyslog:0.1.0 . )
skopeo copy --dest-tls-verify=false docker-daemon:registry.local:31000/githedgehog/rsyslog:0.1.0 docker://registry.local:31000/githedgehog/rsyslog:0.1.0
helm package $SCRIPT_DIR/../third_party/helm/rsyslog-server/charts/rsyslog --version 0.1.1 --app-version 0.1.0 -d $SCRIPT_DIR/../third_party
helm push --insecure-skip-tls-verify $SCRIPT_DIR/../third_party/rsyslog-0.1.1.tgz oci://registry.local:31000/githedgehog/helm-charts
echo

echo "Preparing ntp/chrony for installation..."
docker pull cturra/ntp:latest
docker tag cturra/ntp:latest registry.local:31000/cturra/ntp:latest
skopeo copy --dest-tls-verify=false docker-daemon:registry.local:31000/cturra/ntp:latest docker://registry.local:31000/cturra/ntp:latest
docker push registry.local:31000/cturra/ntp:latest
helm package $SCRIPT_DIR/../third_party/helm/ntp --version 0.0.1 -d $SCRIPT_DIR/../third_party
helm push --insecure-skip-tls-verify $SCRIPT_DIR/../third_party/ntp-0.0.1.tgz oci://registry.local:31000/githedgehog/helm-charts
echo
