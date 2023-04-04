#!/bin/bash
set -e

### START SETTINGS #########
FLATCAR_VERSION=3374.2.4
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

# download OS and UEFI images, etc.pp.
echo "Downloading OS and UEFI images..."
if [ -f ${IMAGE_DIR}/flatcar.img ]; then
    echo "Flatcar OS image already downloaded: ${IMAGE_DIR}/flatcar.img"
    echo "Delete this file if you want to download it again. Skipping..."
else
    echo "Downloading and extracting Flatcar OS image version ${FLATCAR_VERSION}..."
    $WGET -O ${IMAGE_DIR}/flatcar.img.bz2 https://stable.release.flatcar-linux.net/amd64-usr/${FLATCAR_VERSION}/flatcar_production_qemu_uefi_image.img.bz2
    $BUNZIP2 ${IMAGE_DIR}/flatcar.img.bz2
fi
if [ -f ${IMAGE_DIR}/flatcar_efi_code.fd ]; then
    echo "Flatcar EFI code flash drive already downloaded: ${IMAGE_DIR}/flatcar_efi_code.fd"
    echo "Delete this file if you want to download it again. Skipping..."
else
    echo "Downloading Flatcar EFI code flash drive version ${FLATCAR_VERSION}..."
    $WGET -O ${IMAGE_DIR}/flatcar_efi_code.fd https://stable.release.flatcar-linux.net/amd64-usr/3374.2.4/flatcar_production_qemu_uefi_efi_code.fd
fi
if [ -f ${IMAGE_DIR}/flatcar_efi_vars.fd ]; then
    echo "Flatcar EFI variables flash drive already downloaded: ${IMAGE_DIR}/flatcar_efi_vars.fd"
    echo "Delete this file if you want to download it again. Skipping..."
else
    echo "Downloading Flatcar EFI variables flash drive version ${FLATCAR_VERSION}..."
    $WGET -O ${IMAGE_DIR}/flatcar_efi_vars.fd https://stable.release.flatcar-linux.net/amd64-usr/3374.2.4/flatcar_production_qemu_uefi_efi_vars.fd
fi
echo

echo "Downloading SONiC, ONIE and Hedgehog agent images..."
if [ -f ${IMAGE_DIR}/sonic-vs.bin ]; then
    echo "SONiC VS image already downloaded: ${IMAGE_DIR}/sonic-vs.bin"
    echo "Delete this file if you want to download it again. Skipping..."
else
    echo "Downloading SONiC VS image..."
    $WGET -O ${IMAGE_DIR}/sonic-vs.bin https://d.githedgehog.com/sonic/43cfae78-2037-4a4b-b7cf-e3e3c986cc12/sonic-vs.bin
fi
if [ -f ${IMAGE_DIR}/agent ]; then
    echo "Hedgehog agent already downloaded: ${IMAGE_DIR}/agent"
    echo "Delete this file if you want to download it again. Skipping...:"
else
    echo "Downloading Hedgehog agent..."
    $ORAS pull -o ${IMAGE_DIR} ghcr.io/githedgehog/agent:0.2
fi
echo

# let's make a dev folder where we generate everything for the control node
echo -n "Making development folder for control node: "
mkdir -v -p ${SCRIPT_DIR}/../dev/control-node-1
DEV_DIR=$( cd -- "${SCRIPT_DIR}/../dev/control-node-1" &> /dev/null && pwd )
echo ${DEV_DIR}
echo

# copy downloaded images to destination
echo "Copying images to control node development directory..."
cp -v -f ${IMAGE_DIR}/flatcar.img ${DEV_DIR}/os.img
cp -v -f ${IMAGE_DIR}/flatcar_efi_code.fd ${DEV_DIR}/efi_code.fd
cp -v -f ${IMAGE_DIR}/flatcar_efi_vars.fd ${DEV_DIR}/efi_vars.fd
echo

# generate SSH key
echo "Generating SSH key pair for virtual machine..."
$SSH_KEYGEN -t ed25519 -C "core@control-node-1" -f ${DEV_DIR}/core-ssh-key -N "" <<< y
echo

# generate UUID
echo "Generating UUID for virtual machine..."
$UUIDGEN > ${DEV_DIR}/uuid
echo

# create hedgehog conf directory
echo "Creating HEDGEHOG_CONF directory..."
mkdir -v -p $DEV_DIR/hedgehog_conf
echo

# initialize software TPM
echo "Initializing software TPM 2.0 now..."
$SWTPM_SETUP --create-config-files skip-if-exist
if ! [ -f ${HOME}/.config/swtpm_setup.conf ]; then
    echo "ERROR: swtpm config files expected at: ${HOME}/.config/swtpm_setup.conf" 1>&2
    exit 1
fi
mkdir -p $DEV_DIR/tpm
$SWTPM_SETUP \
  --tpm2 \
  --tpmstate $DEV_DIR/tpm \
  --createek \
  --decryption \
  --create-ek-cert \
  --create-platform-cert \
  --create-spk \
  --vmid "control-node-1" \
  --overwrite \
  --display
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
$DOCKER image save -o $DEV_DIR/docker-images/docker-seeder.tar ${DOCKER_REPO:=registry.local:5000/githedgehog/das-boot:latest}
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
echo "Pusing SONiC, ONIE and Hedgehog agent into registry..."
( cd $IMAGE_DIR && $ORAS push registry.local:5000/githedgehog/sonic/x86_64-kvm_x86_64-r0:latest sonic-vs.bin )
( cd $IMAGE_DIR && $ORAS push registry.local:5000/githedgehog/agent/x86_64:latest agent )
echo

# push the CRDs into the local registry
echo "Pusing Agent CRDs into registry..."
( cd $IMAGE_DIR && $HELM pull --version=0.1 oci://ghcr.io/githedgehog/agent-crd )
( cd $IMAGE_DIR && $HELM push agent-crd-0.1.tgz oci://registry.local:5000/githedgehog/helm-charts )
echo
