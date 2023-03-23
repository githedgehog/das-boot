#!/bin/bash
set -e

# NOTE: if you are adding newly generated files, make sure to update DEV_OCI_REPO_CERT_FILES in the Makefile please!

OPENSSL=$(which openssl)
JQ=$(which jq)
IP=$(which ip)

echo "Initializing OCI repository certificates..."
echo

# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# let's make a dev folder where we generate everything for
echo -n "Making development folder: "
mkdir -p ${SCRIPT_DIR}/../dev/oci
DEV_DIR=$( cd -- "${SCRIPT_DIR}/../dev/oci" &> /dev/null && pwd )
echo ${DEV_DIR}
echo

# create CAs
echo "Creating OCI repository CA..."
${OPENSSL} ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/oci-repo-ca-key.pem
${OPENSSL} req -new -nodes -x509 -days 3600 -config ${SCRIPT_DIR}/openssl.cnf -extensions ca_cert -key ${DEV_DIR}/oci-repo-ca-key.pem -out ${DEV_DIR}/oci-repo-ca-cert.pem -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=OCI Repository CA"
echo "Server CA created:"
echo "- ${DEV_DIR}/oci-repo-ca-key.pem"
echo "- ${DEV_DIR}/oci-repo-ca-cert.pem"
echo

# create a server cert
echo "Creating certs..."
SANS="DNS:localhost, DNS:registry.local, IP:127.0.0.1, IP:10.0.2.100"
${OPENSSL} ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/server-key.pem
${OPENSSL} req -new -nodes -x509 -days 360 \
  -CAkey ${DEV_DIR}/oci-repo-ca-key.pem -CA ${DEV_DIR}/oci-repo-ca-cert.pem \
  -key ${DEV_DIR}/server-key.pem -out ${DEV_DIR}/server-cert.pem \
  -config ${SCRIPT_DIR}/openssl.cnf -extensions server_cert \
  -addext "subjectAltName = ${SANS}" \
  -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=localhost"
echo "Server cert created - signed by OCI repository CA:"
echo "- ${DEV_DIR}/server-key.pem"
echo "- ${DEV_DIR}/server-cert.pem"
echo "- SANs: ${SANS}"
echo
echo "################################################################################"
echo "# NOTE: YOU MUST IMPORT THIS CA ON YOUR LOCAL SYSTEM FOR HELM PUSH TO WORK!    #"
echo "################################################################################"
echo 
echo "For example, on Fedora based systems run the following:"
echo
echo "sudo trust anchor --store ${DEV_DIR}/oci-repo-ca-cert.pem"
echo