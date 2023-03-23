#!/bin/bash
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
