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

set -e

# NOTE: this should work on newer Fedora systems like this (FCOS and Flatcar included)
# However, it turns out that Flatcar is still using the "old" mechanism:
# https://www.flatcar.org/docs/latest/setup/security/adding-certificate-authorities/
#trust anchor --store /opt/oci-repo-ca-cert.pem

# Copying our CA certs to /etc/ssl/certs
cp -v /opt/oci-repo-ca-cert.pem /etc/ssl/certs

# Updating system wide CA store again
update-ca-certificates

# ensure we don't need to run this again
touch /opt/ca-certs-imported
