#!/bin/bash
set -e

QEMU_SYSTEM_X86_64=$(which qemu-system-x86_64)
TPM2=$(which tpm2)
PYTHON=$(which python3)

# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

if ! [ -d  "${SCRIPT_DIR}/../dev/control-node-1" ]; then
    echo "ERROR: Run ${SCRIPT_DIR}/init_control_node.sh first to initialize control node" 1>&2
    exit 1
fi
DEV_DIR=$( cd -- "${SCRIPT_DIR}/../dev/control-node-1" &> /dev/null && pwd )
echo "Development directory for control-node-1: ${DEV_DIR}"

# QEMU VM Settings
VM_NAME="control-node-1"
VM_UUID=$(<${DEV_DIR}/uuid)
VM_NCPUS=4
VM_MEMORY=8192
SSH_PORT=2201
KUBE_PORT=6443

# ensure the local docker registry is running
$SCRIPT_DIR/run_registry.sh

# run the TPM in the background if it is not already running
runs_tpm=''
if ! [ -f $DEV_DIR/tpm.pid ]; then
    function on_exit() {
        if [ -f $DEV_DIR/tpm.pid ]; then
            kill $(< $DEV_DIR/tpm.pid) &>/dev/null || true
        fi
        if [ -n $python_webserver1_pid ]; then
            kill $python_webserver1_pid
        fi
        if [ -n $python_webserver2_pid ]; then
            kill $python_webserver2_pid
        fi
    }
    $SCRIPT_DIR/run_control_node_tpm.sh &
    trap on_exit EXIT
    runs_tpm='yes'
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

# run two HTTP file servers in the background
# we use these to serve files through the ignition configuration
# - one for serving extra scripts (which would be to cumbersome to maintain inline, and will be on an OEM partition anyways)
# - another one for serving the docker images to import
$PYTHON -m http.server --bind 127.0.0.1 --directory $SCRIPT_DIR/ignition_files 8888 &
python_webserver1_pid=$!
$PYTHON -m http.server --bind 127.0.0.1 --directory $DEV_DIR/docker-images 8889 &
python_webserver2_pid=$!
if [ -z $runs_tpm ]; then
    function on_exit() {
        if [ -n $python_webserver1_pid ]; then
            kill $python_webserver1_pid
        fi
        if [ -n $python_webserver2_pid ]; then
            kill $python_webserver2_pid
        fi
    }
    trap on_exit EXIT
fi
sleep 1

echo
echo "Running control-node-1 VM now..."
echo
echo "You can access this VM through several mechanisms:"
echo "1. Serial port:"
echo "     make access-control-node-serial"
echo "     socat -,rawer,escape=0x1d unix-connect:$DEV_DIR/serial.sock"
echo "     nc -U $DEV_DIR/serial.sock"
echo
echo "2. VNC:"
echo "     make access-control-node-vnc"
echo "     vncviewer unix $DEV_DIR/vnc.sock"
echo
echo "3. SSH:"
echo "     make access-control-node-ssh"
echo "     ssh -i $DEV_DIR/core-ssh-key -p 2201 core@127.0.0.1"
echo
echo "4. kubeconfig:"
echo "     make access-control-node-kubeconfig"
echo "     ssh -i $DEV_DIR/core-ssh-key -p 2201 core@127.0.0.1 \"sudo kubectl config view --raw=true\""
echo
echo "5. QEMU monitor"
echo "     make access-control-node-monitor"
echo "     nc -U $DEV_DIR/monitor.sock"
echo
echo "6. QEMU QNP: $DEV_DIR/qnp.sock"
echo "     nc -U $DEV_DIR/qnp.sock"
echo

$QEMU_SYSTEM_X86_64 \
  -name "$VM_NAME" \
  -uuid "$VM_UUID" \
  -m "$VM_MEMORY" \
  -machine q35,accel=kvm,smm=on -cpu host -smp "$VM_NCPUS" \
  -netdev user,id=eth0,hostfwd=tcp:127.0.0.1:"$SSH_PORT"-:22,hostfwd=tcp:127.0.0.1:"$KUBE_PORT"-:6443,hostname="$VM_NAME",domainname=local,dnssearch=local \
  -device virtio-net-pci,netdev=eth0,mac=0c:20:12:fe:00:00 \
  -netdev socket,id=eth1,udp=127.0.0.1:21011,localaddr=127.0.0.1:21001 \
  -device virtio-net-pci,netdev=eth1,mac=0c:20:12:fe:00:01 \
  -netdev socket,id=eth2,udp=127.0.0.1:21032,localaddr=127.0.0.1:21002 \
  -device virtio-net-pci,netdev=eth2,mac=0c:20:12:fe:00:02 \
  -object rng-random,filename=/dev/urandom,id=rng0 -device virtio-rng-pci,rng=rng0 \
  -chardev socket,id=chrtpm,path="$DEV_DIR/tpm.sock.ctrl" -tpmdev emulator,id=tpm0,chardev=chrtpm -device tpm-tis,tpmdev=tpm0 \
  -fw_cfg name=opt/org.flatcar-linux/config,file="$DEV_DIR/ignition.json" \
  -drive if=virtio,file="$DEV_DIR/os.img" \
  -drive if=pflash,file="$DEV_DIR/efi_code.fd",format=raw,readonly=on \
  -drive if=pflash,file="$DEV_DIR/efi_vars.fd",format=raw \
  -fsdev local,id=hedgehog-conf,security_model=none,readonly=on,path="$DEV_DIR/hedgehog_conf" \
  -device virtio-9p-pci,fsdev=hedgehog-conf,mount_tag=HEDGEHOG_CONF \
  -display none \
  -vga virtio \
  -vnc unix:$DEV_DIR/vnc.sock \
  -serial unix:$DEV_DIR/serial.sock,server,nowait \
  -monitor unix:$DEV_DIR/monitor.sock,server,nowait \
  -qmp unix:$DEV_DIR/qmp.sock,server,nowait \
  -global ICH9-LPC.disable_s3=1
