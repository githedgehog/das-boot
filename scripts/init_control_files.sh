#!/bin/bash
set -e

### START SETTINGS #########
FLATCAR_VERSION=3510.2.1
### END SETTINGS ###########

# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# executable dependencies for this script
BUTANE=$(which butane)
QEMU_SYSTEM_X86_64=$(which qemu-system-x86_64)
SSH_KEYGEN=$(which ssh-keygen)
UUIDGEN=$(which uuidgen)
WGET=$(which wget)
BUNZIP2=$(which bunzip2)
SWTPM_SETUP=$(which swtpm_setup)
DOCKER=$(which docker)
KUBECTL=$(which kubectl)
HELM=$(which helm)
ORAS=$(which oras)

# let's make a dev folder where we are going to store images
echo -n "Making development folder for storing images: "
mkdir -v -p ${SCRIPT_DIR}/../dev/images
IMAGE_DIR=$( cd -- "${SCRIPT_DIR}/../dev/images" &> /dev/null && pwd )
echo ${IMAGE_DIR}
echo

# let's make a dev folder where we generate everything for the control node
echo -n "Making development folder for control node: "
mkdir -v -p ${SCRIPT_DIR}/../dev/control-node-1
DEV_DIR=$( cd -- "${SCRIPT_DIR}/../dev/control-node-1" &> /dev/null && pwd )
echo ${DEV_DIR}
echo

# Ensure OCI CA has been generated before already
mkdir -p $DEV_DIR/docker-images
mkdir -p ${SCRIPT_DIR}/../dev/oci
OCI_CERT_DIR=$( cd -- "${SCRIPT_DIR}/../dev/oci" &> /dev/null && pwd )
if [ ! -f $OCI_CERT_DIR/oci-repo-ca-cert.pem ] ; then
    $SCRIPT_DIR/init_repo_certs.sh
else
    echo "Copying OCI CA cert to the same directory as well:"
    cp -v $OCI_CERT_DIR/oci-repo-ca-cert.pem $DEV_DIR/docker-images/oci-repo-ca-cert.pem
fi
echo

# now export all docker images that we want to import
echo "Exporting all docker images for import at ignition time..."
$DOCKER image save -o $DEV_DIR/docker-images/docker-seeder.tar ${DOCKER_REPO_SEEDER:=registry.local:5000/githedgehog/das-boot}:latest
$DOCKER image save -o $DEV_DIR/docker-images/docker-registration-controller.tar ${DOCKER_REPO_REGISTRATION_CONTROLLER:=registry.local:5000/githedgehog/das-boot-registration-controller}:latest
echo

# now exporting all seeder secrets
echo "Exporting all seeder secrets for import at ignition time..."
mkdir -p ${SCRIPT_DIR}/../dev/seeder
SEEDER_DEV_DIR=$( cd -- "${SCRIPT_DIR}/../dev/seeder" &> /dev/null && pwd )
$KUBECTL create secret generic das-boot-server-cert --dry-run=client -o yaml --from-file=key.pem=$SEEDER_DEV_DIR/server-key.pem --from-file=cert.pem=$SEEDER_DEV_DIR/server-cert.pem > $DEV_DIR/docker-images/das-boot-server-cert-secret.yaml
$KUBECTL create secret generic das-boot-config-cert --dry-run=client -o yaml --from-file=key.pem=$SEEDER_DEV_DIR/config-key.pem --from-file=cert.pem=$SEEDER_DEV_DIR/config-cert.pem > $DEV_DIR/docker-images/das-boot-config-cert-secret.yaml
$KUBECTL create secret generic das-boot-client-ca --dry-run=client -o yaml --from-file=key.pem=$SEEDER_DEV_DIR/client-ca-key.pem --from-file=cert.pem=$SEEDER_DEV_DIR/client-ca-cert.pem > $DEV_DIR/docker-images/das-boot-client-ca-secret.yaml
$KUBECTL create secret generic das-boot-server-ca --dry-run=client -o yaml --from-file=cert.pem=$SEEDER_DEV_DIR/server-ca-cert.pem > $DEV_DIR/docker-images/das-boot-server-ca-secret.yaml
$KUBECTL create secret generic das-boot-config-ca --dry-run=client -o yaml --from-file=cert.pem=$SEEDER_DEV_DIR/config-ca-cert.pem > $DEV_DIR/docker-images/das-boot-config-ca-secret.yaml
$KUBECTL create secret generic oci-ca --dry-run=client -o yaml --from-file=cert.pem=$OCI_CERT_DIR/oci-repo-ca-cert.pem > $DEV_DIR/docker-images/oci-ca-secret.yaml

# generate ignition config
# we could just pipe everything, but for better debugability, keep it in separate files
echo "Generating ignition config for virtual machine..."
eval "echo \"$(< $SCRIPT_DIR/control-node-ignition.butane.yml)\"" > ${DEV_DIR}/ignition.butane.yml
$BUTANE --files-dir ${DEV_DIR} -o ${DEV_DIR}/ignition.json ${DEV_DIR}/ignition.butane.yml
echo

# preparing all third_party helm/docker installations
echo "Preparing all 3rd party products for installation..."
mkdir -p $DEV_DIR/third_party
echo
echo "Preparing rsyslog for installation..."
( cd $SCRIPT_DIR/../third_party/helm/rsyslog-server && $DOCKER build -t registry.local:5000/githedgehog/rsyslog:0.1.0 . )
$DOCKER push registry.local:5000/githedgehog/rsyslog:0.1.0
$DOCKER image save -o $DEV_DIR/docker-images/docker-syslog.tar registry.local:5000/githedgehog/rsyslog:0.1.0
$HELM package $SCRIPT_DIR/../third_party/helm/rsyslog-server/charts/rsyslog --version 0.1.1 --app-version 0.1.0 -d $DEV_DIR/third_party
$HELM push $DEV_DIR/third_party/rsyslog-0.1.1.tgz oci://registry.local:5000/githedgehog/helm-charts
echo

echo "Preparing ntp/chrony for installation..."
$DOCKER pull cturra/ntp:latest
$DOCKER tag cturra/ntp:latest registry.local:5000/cturra/ntp:latest
$DOCKER push registry.local:5000/cturra/ntp:latest
$DOCKER image save -o $DEV_DIR/docker-images/docker-ntp.tar registry.local:5000/cturra/ntp:latest
$HELM package $SCRIPT_DIR/../third_party/helm/ntp --version 0.0.1 -d $DEV_DIR/third_party
$HELM push $DEV_DIR/third_party/ntp-0.0.1.tgz oci://registry.local:5000/githedgehog/helm-charts
echo

# we'll do this in a subshell so that we can change into the image directory, otherwise the image layer titles will have the full path
echo "Pushing SONiC, ONIE and Hedgehog agent into registry..."
( cd $IMAGE_DIR && $ORAS push registry.local:5000/githedgehog/sonic/x86_64-kvm_x86_64-r0:latest sonic-vs.bin )
( cd $IMAGE_DIR && $ORAS push registry.local:5000/githedgehog/agent/x86_64:latest agent )
echo

# push the CRDs into the local registry
echo "Pushing Agent and Wiring CRDs into registry..."
( cd $IMAGE_DIR && $HELM pull --version=0.3 oci://ghcr.io/githedgehog/agent-crd )
( cd $IMAGE_DIR && $HELM push agent-crd-0.3.tgz oci://registry.local:5000/githedgehog/helm-charts )
( cd $IMAGE_DIR && if [ ! -f wiring-crd-0.4.0.tgz ] ; then $HELM pull --version=0.4.0 oci://ghcr.io/githedgehog/wiring-crd ; fi )
( cd $IMAGE_DIR && $HELM push wiring-crd-0.4.0.tgz oci://registry.local:5000/githedgehog/helm-charts )
( cd $IMAGE_DIR && $HELM pull --version=0.2.0 oci://ghcr.io/githedgehog/fabric-helm )
( cd $IMAGE_DIR && $HELM push fabric-helm-0.2.0.tgz oci://registry.local:5000/githedgehog/helm-charts )
echo
