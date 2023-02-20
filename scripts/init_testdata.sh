#!/bin/sh
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# Some testdata must be initialized because they require root privileges
# and the files cannot be checked in to git unfortunately

# Technically as some of them are unit tests, tney should be rewritten
# as unit tests shouldn't do integration testing.
# However, it would be overkill to extract these few tests into
# integration tests for the time being

# see if we need to run with sudo
SUDO=""
if [ "$(id -u)" -ne "0" ]; then
    SUDO="sudo "
fi

# mknod must be in the path
MKNOD=$(which mknod)
if [ $? -ne 0 ]; then
    echo "ERROR: mknod not in PATH" 1>&2
    exit 1
fi

# pkg/partitions/uevent.go
${SUDO} ${MKNOD} ${SCRIPT_DIR}/pkg/partitions/testdata/DevicePath/dev/loop0 b 7 0
${SUDO} ${MKNOD} ${SCRIPT_DIR}/pkg/partitions/testdata/DevicePath/dev/urandom c 1 0
