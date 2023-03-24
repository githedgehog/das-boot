#!/bin/bash
set -e

### START SETTINGS #########
SWITCH_NAME=${1:-switch}
### END SETTINGS ###########

# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# executable dependencies for this script
UUIDGEN=$(which uuidgen)
WGET=$(which wget)
UNXZ=$(which unxz)
SWTPM_SETUP=$(which swtpm_setup)

# let's make a dev folder where we are going to store images
echo -n "Making development folder for storing images: "
mkdir -v -p ${SCRIPT_DIR}/../dev/images
IMAGE_DIR=$( cd -- "${SCRIPT_DIR}/../dev/images" &> /dev/null && pwd )
echo ${IMAGE_DIR}
echo

# download OS and UEFI images, etc.pp.
# TODO: the links below are from my personal ONIE builds which I Uploaded to Google drive.
# Once we build HONIE, we should replace these with dedicated public release links.
echo "Downloading OS and UEFI images..."
if [ -f ${IMAGE_DIR}/onie-kvm_x86_64.qcow2 ]; then
    echo "ONIE kvm_x86_64 image already downloaded: ${IMAGE_DIR}/onie-kvm_x86_64.qcow2.xz"
    echo "Delete this file if you want to download it again. Skipping..."
else
    echo "Downloading and extracting ONIE kvm_x86_64 image..."
    $WGET -O ${IMAGE_DIR}/onie-kvm_x86_64.qcow2.xz https://drive.google.com/file/d/1hHDBYSk_vbPvwt68nb_e9qFzut80Hsg6/view?usp=share_link
    $UNXZ ${IMAGE_DIR}/onie-kvm_x86_64.qcow2.xz
fi
if [ -f ${IMAGE_DIR}/onie_efi_code.fd ]; then
    echo "ONIE EFI code flash drive already downloaded: ${IMAGE_DIR}/onie_efi_code.fd"
    echo "Delete this file if you want to download it again. Skipping..."
else
    echo "Downloading ONIE EFI code flash drive..."
    $WGET -O ${IMAGE_DIR}/onie_efi_code.fd https://drive.google.com/file/d/1eWs37uWarVhvclv3XmjHfo9Eux8HMa8E/view?usp=share_link
fi
if [ -f ${IMAGE_DIR}/onie_efi_vars.fd ]; then
    echo "Flatcar ONIE EFI variables flash drive already downloaded: ${IMAGE_DIR}/onie_efi_vars.fd"
    echo "Delete this file if you want to download it again. Skipping..."
else
    echo "Downloading ONIE EFI variables flash drive..."
    $WGET -O ${IMAGE_DIR}/onie_efi_vars.fd https://drive.google.com/file/d/1Jc7Twu5JY7RIkOCl5hbxrj9AakotAC5c/view?usp=share_link
fi
echo

# let's make a dev folder where we generate everything for the switch
echo -n "Making development folder for switch $SWITCH_NAME: "
mkdir -v -p ${SCRIPT_DIR}/../dev/$SWITCH_NAME
DEV_DIR=$( cd -- "${SCRIPT_DIR}/../dev/$SWITCH_NAME" &> /dev/null && pwd )
echo ${DEV_DIR}
echo

# copy downloaded images to destination
echo "Copying images to development directory for switch $SWITCH_NAME..."
cp -v -f ${IMAGE_DIR}/onie-kvm_x86_64.qcow2 ${DEV_DIR}/os.img
cp -v -f ${IMAGE_DIR}/onie_efi_code.fd ${DEV_DIR}/efi_code.fd
cp -v -f ${IMAGE_DIR}/onie_efi_vars.fd ${DEV_DIR}/efi_vars.fd
echo

# generate UUID
echo "Generating UUID for virtual machine..."
$UUIDGEN > ${DEV_DIR}/uuid
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
  --vmid "$SWITCH_NAME" \
  --overwrite \
  --display
echo
