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

# pkg/partitions/uevent_test.go
mkdir -p ${SCRIPT_DIR}/pkg/partitions/testdata/DevicePath/dev
${SUDO} ${MKNOD} ${SCRIPT_DIR}/pkg/partitions/testdata/DevicePath/dev/loop0 b 7 0
${SUDO} ${MKNOD} ${SCRIPT_DIR}/pkg/partitions/testdata/DevicePath/dev/urandom c 1 0

# pkg/partitions/device_test.go
mkdir -p ${SCRIPT_DIR}/pkg/partitions/testdata/ensureDevicePath/dev
${SUDO} ${MKNOD} ${SCRIPT_DIR}/pkg/partitions/testdata/ensureDevicePath/dev/loop0 b 7 0
