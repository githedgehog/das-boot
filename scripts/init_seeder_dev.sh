#!/bin/bash
set -e

# NOTE: if you are adding newly generated files, make sure to update DEV_FILES in the Makefile please!

echo "Initializing seeder development environment..."
echo

# path where this script resides
SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

# let's make a dev folder where we generate everything for
echo -n "Making development folder: "
mkdir -p ${SCRIPT_DIR}/../dev
DEV_DIR=$( cd -- "${SCRIPT_DIR}/../dev/" &> /dev/null && pwd )
echo ${DEV_DIR}
echo

# create CAs
echo "Creating CAs..."
openssl ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/server-ca-key.pem
openssl req -new -nodes -x509 -days 3600 -config ${SCRIPT_DIR}/openssl.cnf -extensions ca_cert -key ${DEV_DIR}/server-ca-key.pem -out ${DEV_DIR}/server-ca-cert.pem -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=DAS BOOT Server CA"
echo "Server CA created:"
echo "- ${DEV_DIR}/server-ca-key.pem"
echo "- ${DEV_DIR}/server-ca-cert.pem"
openssl ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/client-ca-key.pem
openssl req -new -nodes -x509 -days 3600 -config ${SCRIPT_DIR}/openssl.cnf -extensions ca_cert -key ${DEV_DIR}/client-ca-key.pem -out ${DEV_DIR}/client-ca-cert.pem -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=DAS BOOT Client CA"
echo "Client CA created:"
echo "- ${DEV_DIR}/client-ca-key.pem"
echo "- ${DEV_DIR}/client-ca-cert.pem"
openssl ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/config-ca-key.pem
openssl req -new -nodes -x509 -days 3600 -config ${SCRIPT_DIR}/openssl.cnf -extensions ca_cert -key ${DEV_DIR}/config-ca-key.pem -out ${DEV_DIR}/config-ca-cert.pem -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=DAS BOOT Config Signatures CA"
echo "Config Signature CA created:"
echo "- ${DEV_DIR}/config-ca-key.pem"
echo "- ${DEV_DIR}/config-ca-cert.pem"
echo

# create a server cert
echo "Creating certs..."
SANS="DNS:localhost, DNS:das-boot.hedgehog.svc.cluster.local, IP:127.0.0.1, IP:192.168.42.11"
openssl ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/server-key.pem
openssl req -new -nodes -x509 -days 360 \
  -CAkey ${DEV_DIR}/server-ca-key.pem -CA ${DEV_DIR}/server-ca-cert.pem \
  -key ${DEV_DIR}/server-key.pem -out ${DEV_DIR}/server-cert.pem \
  -config ${SCRIPT_DIR}/openssl.cnf -extensions server_cert \
  -addext "subjectAltName = ${SANS}" \
  -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=localhost"
echo "Server cert created - signed by Server CA:"
echo "- ${DEV_DIR}/server-key.pem"
echo "- ${DEV_DIR}/server-cert.pem"
echo "- SANs: ${SANS}"

# create a config signing cert
openssl ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/config-key.pem
openssl req -new -nodes -x509 -days 360 \
  -CAkey ${DEV_DIR}/config-ca-key.pem -CA ${DEV_DIR}/config-ca-cert.pem \
  -key ${DEV_DIR}/config-key.pem -out ${DEV_DIR}/config-cert.pem \
  -config ${SCRIPT_DIR}/openssl.cnf -extensions code_sign_cert \
  -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=Embedded Config Generator"
echo "Config Signature cert created - signed by Config Signature CA:"
echo "- ${DEV_DIR}/config-key.pem"
echo "- ${DEV_DIR}/config-cert.pem"
echo

# now create a config file
echo "Creating config file for seeder..."
cat << EOF > ${DEV_DIR}/seeder.yaml
servers:
  insecure:
    # adjust these as needed, they should be IPv6 link-local addresses
    addresses:
$(for i in $(ip addr | grep inet6 | grep "fe80" | awk '{ print $2 }' | sed 's#/[[:digit:]]\+##'); do echo "      - $i"; done)
  secure:
    # adjust these as needed, they should be the "management vlan" IPs
    addresses:
$(for i in $(ip addr | grep "inet " | awk '{ print $2 }' | sed 's#/[[:digit:]]\+##'); do echo "      - $i"; done)
    client_ca: ${DEV_DIR}/client-ca-cert.pem
    server_key: ${DEV_DIR}/server-key.pem
    server_cert: ${DEV_DIR}/server-cert.pem
embedded_config_generator:
  config_signature_key: ${DEV_DIR}/config-key.pem
  config_signature_cert: ${DEV_DIR}/config-cert.pem
installer_settings:
  server_ca: ${DEV_DIR}/server-ca-cert.pem
  config_signature_ca: ${DEV_DIR}/config-ca-cert.pem
  # adjust these as needed, should match one of the SANs of the server cert
  secure_server_name: localhost
  # adjust all these to your dev needs
  dns_servers:
    - 127.0.0.1
  ntp_servers:
    - 127.0.0.1
  syslog_servers:
    - 127.0.0.1
EOF

echo -n "Config written to: "
echo ${DEV_DIR}/seeder.yaml
echo