#!/bin/bash

SCRIPT_DIR=$( cd -- "$( dirname -- "${BASH_SOURCE[0]}" )" &> /dev/null && pwd )

mkdir -p ${SCRIPT_DIR}/../dev

DEV_DIR=${SCRIPT_DIR}/../dev

# create CAs
openssl ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/server-ca-key.pem
openssl req -new -nodes -x509 -days 3600 -config ${SCRIPT_DIR}/openssl.cnf -extensions ca_cert -key ${DEV_DIR}/server-ca-key.pem -out ${DEV_DIR}/server-ca-cert.pem -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=DAS BOOT Server CA"
openssl ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/client-ca-key.pem
openssl req -new -nodes -x509 -days 3600 -config ${SCRIPT_DIR}/openssl.cnf -extensions ca_cert -key ${DEV_DIR}/client-ca-key.pem -out ${DEV_DIR}/client-ca-cert.pem -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=DAS BOOT Client CA"
openssl ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/config-ca-key.pem
openssl req -new -nodes -x509 -days 3600 -config ${SCRIPT_DIR}/openssl.cnf -extensions ca_cert -key ${DEV_DIR}/config-ca-key.pem -out ${DEV_DIR}/config-ca-cert.pem -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=DAS BOOT Config Signatures CA"

# create a server cert
openssl ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/server-key.pem
openssl req -new -nodes -x509 -days 360 \
  -CAkey ${DEV_DIR}/server-ca-key.pem -CA ${DEV_DIR}/server-ca-cert.pem \
  -key ${DEV_DIR}/server-key.pem -out ${DEV_DIR}/server-cert.pem \
  -config ${SCRIPT_DIR}/openssl.cnf -extensions server_cert \
  -addext "subjectAltName = DNS:localhost, DNS:das-boot.hedgehog.svc.cluster.local, IP:127.0.0.1, IP:192.168.42.11" \
  -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=localhost"

# create a config signing cert
openssl ecparam -name prime256v1 -genkey -noout -out ${DEV_DIR}/config-key.pem
openssl req -new -nodes -x509 -days 360 \
  -CAkey ${DEV_DIR}/config-ca-key.pem -CA ${DEV_DIR}/config-ca-cert.pem \
  -key ${DEV_DIR}/config-key.pem -out ${DEV_DIR}/config-cert.pem \
  -config ${SCRIPT_DIR}/openssl.cnf -extensions code_sign_cert \
  -subj "/C=US/ST=Washington/L=Seattle/O=Hedgehog SONiC Foundation/CN=Embedded Config Generator"

