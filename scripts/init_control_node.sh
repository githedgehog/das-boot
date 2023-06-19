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
    $WGET -O ${IMAGE_DIR}/flatcar_efi_code.fd https://stable.release.flatcar-linux.net/amd64-usr/${FLATCAR_VERSION}/flatcar_production_qemu_uefi_efi_code.fd
fi
if [ -f ${IMAGE_DIR}/flatcar_efi_vars.fd ]; then
    echo "Flatcar EFI variables flash drive already downloaded: ${IMAGE_DIR}/flatcar_efi_vars.fd"
    echo "Delete this file if you want to download it again. Skipping..."
else
    echo "Downloading Flatcar EFI variables flash drive version ${FLATCAR_VERSION}..."
    $WGET -O ${IMAGE_DIR}/flatcar_efi_vars.fd https://stable.release.flatcar-linux.net/amd64-usr/${FLATCAR_VERSION}/flatcar_production_qemu_uefi_efi_vars.fd
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
    $ORAS pull -o ${IMAGE_DIR} ghcr.io/githedgehog/agent:0.3
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
