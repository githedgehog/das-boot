#!/bin/bash
set -e

SWITCH_NAME=${1:-switch}

SWTPM=$(which swtpm)

# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

if ! [ -d  "${SCRIPT_DIR}/../dev/$SWITCH_NAME/tpm" ]; then
    echo "ERROR: Run ${SCRIPT_DIR}/init_switch.sh $SWITCH_NAME first to initialize the switch" 1>&2
    exit 1
fi
DEV_DIR=$( cd -- "${SCRIPT_DIR}/../dev/$SWITCH_NAME" &> /dev/null && pwd )
echo "Development directory for $SWITCH_NAME: ${DEV_DIR}"

# cleanup sockets on exit
function on_exit() {
    rm -v -f $DEV_DIR/tpm.sock.ctrl || true
    rm -v -f $DEV_DIR/tpm.sock || true
    rm -v -f $DEV_DIR/tpm.pid || true
}
trap on_exit EXIT

echo "Running software TPM 2.0 now..."
echo
echo "You can access this TPM through two ways:"
echo "1. In QEMU:"
echo "   use control socket $DEV_DIR/tpm.sock.ctrl"
echo
echo "2. direct access to the TPM with tpm2-tools, run this in your shell:"
echo "   NOTE: this will not work when QEMU has taken over"
echo
echo "   export TPM2TOOLS_TCTI=swtpm:path=$DEV_DIR/tpm.sock"
echo
$SWTPM socket \
  --tpm2 \
  --tpmstate dir=$DEV_DIR/tpm \
  --ctrl type=unixio,path=$DEV_DIR/tpm.sock.ctrl \
  --server type=unixio,path=$DEV_DIR/tpm.sock \
  --pid file=$DEV_DIR/tpm.pid \
  --log level=1 \
  --flags startup-clear \
  $@
