#!/bin/bash
set -e

DEFAULT_NETDEVS=(
    "devid=eth0 mac=0c:20:12:fe:01:01 local_port=127.0.0.1:21001 dest_port=127.0.0.1:21000"
)

# we cannot pass bash arrays, so we will have to parse that
# taking "devid=" as the indicator that this is a new entry
tmp="$NETDEVS"
NETDEVS=()
if [ -n "$tmp" ]; then
    # count=-1
    for i in $tmp; do
        if [[ "$i" = devid=* ]]; then
            ((count=$count+1))
        fi
        NETDEVS[$count]="${NETDEVS[$count]} $i"
    done
else
    NETDEVS=("${DEFAULT_NETDEVS[@]}")
fi

### START SETTINGS #########
SWITCH_NAME=${1:-switch}
VM_NCPUS=${VM_NCPUS:-1}
VM_MEMORY=${VM_MEMORY:-4096}
# NETDEVS - see above
### END SETTINGS ###########

echo "Configured QEMU network devices:"
qemu_devices=""
for netdev in "${NETDEVS[@]}"; do
    eval $netdev
    echo "- name: $devid, MAC: $mac, Connection: $local_port -> $dest_port"
    qemu_devices="${qemu_devices} -netdev socket,id=$devid,udp=$dest_port,localaddr=$local_port -device virtio-net-pci,netdev=$devid,mac=$mac"
done

QEMU_SYSTEM_X86_64=$(which qemu-system-x86_64)
TPM2=$(which tpm2)

# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

if ! [ -d  "${SCRIPT_DIR}/../dev/$SWITCH_NAME" ]; then
    echo "ERROR: Run ${SCRIPT_DIR}/init_switch.sh $SWITCH_NAME first to initialize switch" 1>&2
    exit 1
fi
DEV_DIR=$( cd -- "${SCRIPT_DIR}/../dev/$SWITCH_NAME" &> /dev/null && pwd )
echo "Development directory for switch $SWITCH_NAME: ${DEV_DIR}"

# QEMU VM Settings
VM_NAME="$SWITCH_NAME"
VM_UUID=$(<${DEV_DIR}/uuid)

# ensure the local docker registry is running
$SCRIPT_DIR/run_registry.sh

# run the TPM in the background if it is not already running
if ! [ -f $DEV_DIR/tpm.pid ]; then
    function on_exit() {
        if [ -f $DEV_DIR/tpm.pid ]; then
            kill $(< $DEV_DIR/tpm.pid) &>/dev/null || true
        fi
    }
    $SCRIPT_DIR/run_switch_tpm.sh $SWITCH_NAME &
    trap on_exit EXIT
    sleep 1
    if ! [ -S $DEV_DIR/tpm.sock.ctrl ]; then
        echo "ERROR: software TPM failed to start control channel $DEV_DIR/tpm.sock.ctrl" 1>&2
        exit 1
    fi
fi

# This is an ugly workaround in a bug in swtpm:
# If you specify both --server and --ctrl flags for the socket swtpm,
# then it exits if you start with QEMU directly. If you run a command,
# then it will continue to work.
TPM2TOOLS_TCTI="swtpm:path=$DEV_DIR/tpm.sock" $TPM2 startup

echo
echo "Running switch $SWITCH_NAME VM now..."
echo
echo "You can access this VM through several mechanisms:"
echo "1. Serial port:"
echo "     make access-switch-$SWITCH_NAME-serial"
echo "     socat -,rawer,escape=0x1d unix-connect:$DEV_DIR/serial.sock"
echo "     nc -U $DEV_DIR/serial.sock"
echo
echo "2. QEMU monitor"
echo "     make access-switch-$SWITCH_NAME-monitor"
echo "     nc -U $DEV_DIR/monitor.sock"
echo
echo "3. QEMU QNP: $DEV_DIR/qnp.sock"
echo "     nc -U $DEV_DIR/qnp.sock"
echo

$QEMU_SYSTEM_X86_64 \
  -name "$VM_NAME" \
  -uuid "$VM_UUID" \
  -m "$VM_MEMORY" \
  -machine q35,accel=kvm,smm=on -cpu host -smp "$VM_NCPUS" \
  -netdev socket,id=eth1,udp=127.0.0.1:21000,localaddr=127.0.0.1:21001 \
  -device virtio-net-pci,netdev=eth1,mac=0c:20:12:fe:01:01 \
  -object rng-random,filename=/dev/urandom,id=rng0 -device virtio-rng-pci,rng=rng0 \
  -chardev socket,id=chrtpm,path="$DEV_DIR/tpm.sock.ctrl" -tpmdev emulator,id=tpm0,chardev=chrtpm -device tpm-tis,tpmdev=tpm0 \
  -drive if=virtio,file="$DEV_DIR/os.img" \
  -drive if=pflash,file="$DEV_DIR/efi_code.fd",format=raw,readonly=on \
  -drive if=pflash,file="$DEV_DIR/efi_vars.fd",format=raw \
  -display none \
  -vga none \
  -serial unix:$DEV_DIR/serial.sock,server,nowait \
  -monitor unix:$DEV_DIR/monitor.sock,server,nowait \
  -qmp unix:$DEV_DIR/qmp.sock,server,nowait \
  -global ICH9-LPC.disable_s3=1
